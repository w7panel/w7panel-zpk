package logic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/yaml"
)

//go:embed helm_templates/*.tpl
var helmTemplateFS embed.FS

var workflowChartPathPattern = regexp.MustCompile(`^workflow/[1-9][0-9]*/(workflow\.tgz|runs/[0-9a-f]{32}\.tgz)$`)
var workflowRunIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

const workflowConfigAnnotation = "w7.cc/workflow-config-sha256"

type Helm struct{}

// GenerateChart is called when configuration changes, never for individual triggers.
func (h Helm) GenerateChart(config Config) ([]byte, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "zpk-workflow-chart-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	rootDir := filepath.Join(directory, "workflow")
	if err := os.MkdirAll(filepath.Join(rootDir, "templates"), 0o755); err != nil {
		return nil, err
	}
	if err := h.generateChartYaml(rootDir); err != nil {
		return nil, err
	}
	if err := h.generateValuesYaml(rootDir, config); err != nil {
		return nil, err
	}
	for _, generate := range []func(string) error{
		h.generateServiceAccountYaml,
		h.generateConfigMapYaml,
		h.generateRoleYaml,
		h.generateRoleBindingYaml,
		h.generateSecretYaml,
		h.generateJobYaml,
	} {
		if err := generate(rootDir); err != nil {
			return nil, err
		}
	}
	workflowChart, err := loader.LoadDir(rootDir)
	if err != nil {
		return nil, err
	}
	configDigest, err := h.configDigest(config)
	if err != nil {
		return nil, err
	}
	if workflowChart.Metadata.Annotations == nil {
		workflowChart.Metadata.Annotations = map[string]string{}
	}
	workflowChart.Metadata.Annotations[workflowConfigAnnotation] = configDigest
	chartArchivePath, err := chartutil.Save(workflowChart, directory)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(chartArchivePath)
}

// ChartPath is stable for a repository; only saving configuration replaces this file.
func (h Helm) ChartPath(repositoryID int32) (string, error) {
	if repositoryID < 1 {
		return "", errors.New("仓库 ID 无效")
	}
	return fmt.Sprintf("workflow/%d/workflow.tgz", repositoryID), nil
}

// RunChartPath is derived from the run ID, never persisted in the database.
func (h Helm) RunChartPath(repositoryID int32, runID string) (string, error) {
	if repositoryID < 1 || !workflowRunIDPattern.MatchString(runID) {
		return "", errors.New("工作流执行标识无效")
	}
	return fmt.Sprintf("workflow/%d/runs/%s.tgz", repositoryID, runID), nil
}

// The digest checks that the current package agrees with the transaction's rules.
// It is not used as a directory name. Runtime credentials are excluded.
func (h Helm) configDigest(config Config) (string, error) {
	tasks := append([]Task(nil), config.Tasks...)
	for i := range tasks {
		tasks[i].K8sConfig = ""
		if tasks[i].Name == "" {
			tasks[i].Name = tasks[i].K8sAppName
		}
		if tasks[i].DeployType == 0 {
			tasks[i].DeployType = 1
		}
	}
	slices.SortFunc(tasks, func(a, b Task) int { return strings.Compare(a.ID, b.ID) })
	content, err := json.Marshal(struct {
		Version string `json:"version"`
		Tasks   []Task `json:"tasks"`
	}{Version: "workflow-explicit-cluster-auth-v5", Tasks: tasks})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(content)), nil
}

func (h Helm) SaveChart(repositoryID int32, config Config) (string, error) {
	relative, err := h.ChartPath(repositoryID)
	if err != nil {
		return "", err
	}
	content, err := h.GenerateChart(config)
	if err != nil {
		return "", err
	}
	if err := h.writeChart(relative, content, true); err != nil {
		return "", err
	}
	return relative, nil
}

// writeChart atomically publishes a complete file. Run packages must never be overwritten.
func (h Helm) writeChart(relativeChartPath string, content []byte, replaceExisting bool) error {
	filename, err := h.chartFilePath(relativeChartPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(filename), ".workflow-*.tgz")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if replaceExisting {
		return os.Rename(temporary, filename)
	}
	return os.Link(temporary, filename)
}

// PinChart copies the saved package, without regenerating it, before enqueuing a run.
func (h Helm) PinChart(repositoryID int32, runID string, config Config) error {
	runChartPath, err := h.RunChartPath(repositoryID, runID)
	if err != nil {
		return err
	}
	if runChart, err := h.LoadChart(runChartPath); err == nil {
		return h.checkChartConfig(runChart, config)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	currentChartPath, err := h.ChartPath(repositoryID)
	if err != nil {
		return err
	}
	currentChartFile, err := h.chartFilePath(currentChartPath)
	if err != nil {
		return err
	}
	chartArchiveBytes, err := os.ReadFile(currentChartFile)
	if err != nil {
		return fmt.Errorf("读取当前工作流 Helm 包，请重新保存任务配置: %w", err)
	}
	workflowChart, err := loader.LoadArchive(bytes.NewReader(chartArchiveBytes))
	if err != nil {
		return fmt.Errorf("读取工作流 Helm 包: %w", err)
	}
	if err := h.checkChartConfig(workflowChart, config); err != nil {
		return err
	}
	if err := h.writeChart(runChartPath, chartArchiveBytes, false); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return err
		}
		runChart, err := h.LoadChart(runChartPath)
		if err != nil {
			return err
		}
		return h.checkChartConfig(runChart, config)
	}
	return nil
}

func (h Helm) checkChartConfig(workflowChart *chart.Chart, config Config) error {
	expectedConfigDigest, err := h.configDigest(config)
	if err != nil {
		return err
	}
	if workflowChart.Metadata.Annotations[workflowConfigAnnotation] != expectedConfigDigest {
		return errors.New("工作流 Helm 包与任务配置不一致，请重新保存任务配置")
	}
	return nil
}

func (h Helm) RemoveRunChart(repositoryID int32, runID string) error {
	relative, err := h.RunChartPath(repositoryID, runID)
	if err != nil {
		return err
	}
	filename, err := h.chartFilePath(relative)
	if err != nil {
		return err
	}
	if err := os.Remove(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

type runChartFile struct {
	RepositoryID int32
	RunID        string
	ModifiedAt   time.Time
}

// runCharts lists only temporary execution packages, leaving current and legacy packages alone.
func (h Helm) runCharts() ([]runChartFile, error) {
	base := facade.GetConfig().GetString("setting.depot.storage.local.path")
	if strings.TrimSpace(base) == "" {
		return nil, errors.New("未配置制品本地存储目录")
	}
	root := filepath.Join(base, "Helm", "workflow")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []runChartFile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id, err := strconv.ParseInt(entry.Name(), 10, 32)
		if err != nil || id < 1 || strconv.FormatInt(id, 10) != entry.Name() {
			continue
		}
		runs, err := os.ReadDir(filepath.Join(root, entry.Name(), "runs"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			if run.IsDir() || !strings.HasSuffix(run.Name(), ".tgz") {
				continue
			}
			runID := strings.TrimSuffix(run.Name(), ".tgz")
			if !workflowRunIDPattern.MatchString(runID) {
				continue
			}
			info, err := run.Info()
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, err
			}
			files = append(files, runChartFile{RepositoryID: int32(id), RunID: runID, ModifiedAt: info.ModTime()})
		}
	}
	return files, nil
}

func (h Helm) chartFilePath(relative string) (string, error) {
	if !workflowChartPathPattern.MatchString(relative) {
		return "", fmt.Errorf("工作流 Helm 包路径无效")
	}
	base := facade.GetConfig().GetString("setting.depot.storage.local.path")
	if strings.TrimSpace(base) == "" {
		return "", fmt.Errorf("未配置制品本地存储目录")
	}
	return filepath.Join(base, "Helm", filepath.FromSlash(relative)), nil
}

func (h Helm) LoadChart(relative string) (*chart.Chart, error) {
	filename, err := h.chartFilePath(relative)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取工作流 Helm 包: %w", err)
	}
	workflowChart, err := loader.LoadArchive(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("读取任务 Helm 包: %w", err)
	}
	return workflowChart, nil
}

func (h Helm) generateChartYaml(rootDir string) error {
	return h.writeTemplateFile(rootDir, "Chart.yaml", "Chart.yaml.tpl")
}

func (h Helm) generateValuesYaml(rootDir string, config Config) error {
	stages := []int{}
	for _, task := range config.Tasks {
		if !slices.Contains(stages, task.Stage) {
			stages = append(stages, task.Stage)
		}
	}
	slices.Sort(stages)
	tasks := append([]Task(nil), config.Tasks...)
	for i := range tasks {
		tasks[i].K8sConfig = ""
	}
	content, err := helmTemplateFS.ReadFile("helm_templates/values.yaml.tpl")
	if err != nil {
		return err
	}
	valuesTemplate, err := template.New("values.yaml.tpl").Funcs(template.FuncMap{
		"toYaml": func(value interface{}) (string, error) {
			data, err := yaml.Marshal(value)
			return strings.TrimSuffix(string(data), "\n"), err
		},
		"indent": func(spaces int, value string) string {
			prefix := strings.Repeat(" ", spaces)
			return prefix + strings.ReplaceAll(value, "\n", "\n"+prefix)
		},
	}).Parse(string(content))
	if err != nil {
		return fmt.Errorf("解析工作流 values.yaml: %w", err)
	}
	var buffer bytes.Buffer
	if err := valuesTemplate.Execute(&buffer, struct {
		Tasks  []Task
		Stages []int
	}{Tasks: tasks, Stages: stages}); err != nil {
		return fmt.Errorf("生成工作流 values.yaml: %w", err)
	}
	return os.WriteFile(filepath.Join(rootDir, "values.yaml"), buffer.Bytes(), 0o600)
}

func (h Helm) generateServiceAccountYaml(rootDir string) error {
	return h.writeTemplateFile(filepath.Join(rootDir, "templates"), "serviceaccount.yaml", "serviceaccount.yaml.tpl")
}

func (h Helm) generateConfigMapYaml(rootDir string) error {
	return h.writeTemplateFile(filepath.Join(rootDir, "templates"), "configmap.yaml", "configmap.yaml.tpl")
}

func (h Helm) generateRoleYaml(rootDir string) error {
	return h.writeTemplateFile(filepath.Join(rootDir, "templates"), "role.yaml", "role.yaml.tpl")
}

func (h Helm) generateRoleBindingYaml(rootDir string) error {
	return h.writeTemplateFile(filepath.Join(rootDir, "templates"), "rolebinding.yaml", "rolebinding.yaml.tpl")
}

func (h Helm) generateSecretYaml(rootDir string) error {
	return h.writeTemplateFile(filepath.Join(rootDir, "templates"), "secret.yaml", "secret.yaml.tpl")
}

func (h Helm) generateJobYaml(rootDir string) error {
	return h.writeTemplateFile(filepath.Join(rootDir, "templates"), "job.yaml", "job.yaml.tpl")
}

func (h Helm) writeTemplateFile(rootDir, name, templateName string) error {
	content, err := helmTemplateFS.ReadFile(path.Join("helm_templates", templateName))
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(rootDir, name), content, 0o600)
}

// Install owns the release lifecycle; collect runs before Helm removes its resources.
func (h Helm) Install(ctx context.Context, run entity.RegistryRepositoryDeployRuleMatchLog, config Config, values map[string]interface{}, collectTaskResults func(kubernetes.Interface, *release.Release) error) error {
	// Use the service account, never the operator's local kubeconfig.
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clusterConfigGetter := &clusterGetter{config: restConfig, namespace: executionNamespace}
	helmConfig := new(action.Configuration)
	if err = helmConfig.Init(clusterConfigGetter, executionNamespace, "secret", func(format string, args ...interface{}) { slog.Debug(fmt.Sprintf(format, args...)) }); err != nil {
		return err
	}
	releaseName := "wf-" + run.WorkflowRunID
	if _, err = helmConfig.Releases.Last(releaseName); err == nil {
		return errors.New("执行实例已经存在，拒绝重放")
	} else if !errors.Is(err, driver.ErrReleaseNotFound) {
		return err
	}
	// Regular resources and Helm release metadata are transient. Hook Jobs retain
	// logs for their TTL; execution snapshots remain in the existing log table.
	defer func() {
		uninstall := action.NewUninstall(helmConfig)
		uninstall.Timeout = time.Minute
		if _, cleanupErr := uninstall.Run(releaseName); cleanupErr != nil && !errors.Is(cleanupErr, driver.ErrReleaseNotFound) {
			slog.Error("cleanup workflow release", "run", run.WorkflowRunID, "error", cleanupErr)
		}
	}()
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	var helmRelease *release.Release
	if collectTaskResults != nil {
		defer func() {
			if err := collectTaskResults(client, helmRelease); err != nil {
				slog.Error("collect workflow task results", "run", run.WorkflowRunID, "error", err)
			}
		}()
	}

	chartPath, err := h.RunChartPath(run.RepositoryID, run.WorkflowRunID)
	if err != nil {
		return err
	}
	workflowChart, err := h.LoadChart(chartPath)
	if err != nil {
		return err
	}
	if err := h.checkChartConfig(workflowChart, config); err != nil {
		return err
	}
	install := action.NewInstall(helmConfig)
	install.ReleaseName = releaseName
	install.Namespace = executionNamespace
	install.Timeout = 2 * time.Hour
	install.Wait = true
	install.WaitForJobs = true
	helmRelease, err = install.RunWithContext(ctx, workflowChart, values)
	if err != nil && ctx.Err() != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if cleanupErr := client.BatchV1().Jobs(executionNamespace).DeleteCollection(cleanupCtx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "w7.cc/workflow-run=" + releaseName}); cleanupErr != nil {
			slog.Error("stop timed out workflow", "run", run.WorkflowRunID, "error", cleanupErr)
		}
	}
	return err
}

// clusterGetter restricts Helm to the current service account configuration.
type clusterGetter struct {
	config    *rest.Config
	namespace string
}

func (g *clusterGetter) ToRESTConfig() (*rest.Config, error) { return rest.CopyConfig(g.config), nil }
func (g *clusterGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	client, err := discovery.NewDiscoveryClientForConfig(g.config)
	if err != nil {
		return nil, err
	}
	return memory.NewMemCacheClient(client), nil
}
func (g *clusterGetter) ToRESTMapper() (meta.RESTMapper, error) {
	client, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(client), nil
}
func (g *clusterGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveClientConfig(clientcmdapi.Config{CurrentContext: "workflow", Contexts: map[string]*clientcmdapi.Context{"workflow": {Cluster: "workflow", AuthInfo: "workflow", Namespace: g.namespace}}, Clusters: map[string]*clientcmdapi.Cluster{"workflow": {Server: g.config.Host, CertificateAuthority: g.config.CAFile, CertificateAuthorityData: g.config.CAData}}, AuthInfos: map[string]*clientcmdapi.AuthInfo{"workflow": {Token: g.config.BearerToken, TokenFile: g.config.BearerTokenFile}}}, "workflow", &clientcmd.ConfigOverrides{}, nil)
}
