package logic

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/kube"
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
	"slices"
	"strings"
	"text/template"
	"time"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"sigs.k8s.io/yaml"
)

//go:embed helm_templates/*.tpl
var helmTemplateFS embed.FS

type Helm struct{}

// GenerateChart builds an in-memory chart from the immutable execution snapshot.
func (h Helm) GenerateChart(config Config) (*chart.Chart, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	values, err := h.generateValuesYaml(config)
	if err != nil {
		return nil, err
	}
	files := []*loader.BufferedFile{{Name: "values.yaml", Data: values}}
	entries, err := helmTemplateFS.ReadDir("helm_templates")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "values.yaml.tpl" {
			continue
		}
		content, err := helmTemplateFS.ReadFile("helm_templates/" + name)
		if err != nil {
			return nil, err
		}
		name = strings.TrimSuffix(name, ".tpl")
		if name != "Chart.yaml" {
			name = "templates/" + name
		}
		files = append(files, &loader.BufferedFile{Name: name, Data: content})
	}
	return loader.LoadFiles(files)
}

func (h Helm) generateValuesYaml(config Config) ([]byte, error) {
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
		return nil, err
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
		return nil, fmt.Errorf("解析工作流 values.yaml: %w", err)
	}
	var buffer bytes.Buffer
	if err := valuesTemplate.Execute(&buffer, struct {
		Tasks      []Task
		Stages     []int
		Repository RepositoryConfig
	}{Tasks: tasks, Stages: stages, Repository: config.Repository}); err != nil {
		return nil, fmt.Errorf("生成工作流 values.yaml: %w", err)
	}
	return buffer.Bytes(), nil
}

// Install owns the release lifecycle; collect runs before Helm removes its resources.
func (h Helm) Install(ctx context.Context, run entity.RegistryRepositoryDeployRuleMatchLog, config Config, values map[string]interface{}, collectTaskResults func(kubernetes.Interface, *release.Release, bool) error) error {
	// Use the service account, never the operator's local kubeconfig.
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return err
	}
	clusterConfigGetter := &clusterGetter{config: restConfig, namespace: executionNamespace}
	helmConfig := new(action.Configuration)
	if err = helmConfig.Init(clusterConfigGetter, executionNamespace, "memory", func(format string, args ...interface{}) { slog.Debug(fmt.Sprintf(format, args...)) }); err != nil {
		return err
	}
	releaseName := "wf-" + run.WorkflowRunID
	// Release state stays in memory; ordinary resources are removed after results
	// are collected. Hook Jobs are removed by their short Kubernetes TTL.
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
		helmConfig.KubeClient = &workflowKubeClient{
			Client: helmConfig.KubeClient.(*kube.Client),
			collectTaskResults: func() error {
				return collectTaskResults(client, nil, false)
			},
		}
		defer func() {
			if err := collectTaskResults(client, helmRelease, true); err != nil {
				slog.Error("collect workflow task results", "run", run.WorkflowRunID, "error", err)
			}
		}()
	}

	workflowChart, err := h.GenerateChart(config)
	if err != nil {
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

// Collect each completed stage before later stages can outlive its Job TTL.
type workflowKubeClient struct {
	*kube.Client
	collectTaskResults func() error
}

func (c *workflowKubeClient) WatchUntilReady(resources kube.ResourceList, timeout time.Duration) error {
	err := c.Client.WatchUntilReady(resources, timeout)
	if collectErr := c.collectTaskResults(); collectErr != nil {
		slog.Error("collect workflow stage results", "error", collectErr)
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
