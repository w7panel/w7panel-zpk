package logic

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

const (
	RegistryStorageTypeFilesystem = "filesystem"
	RegistryStorageTypeS3         = "s3"
	registryStorageMountPath      = "/var/lib/registry"
)

type RegistryStorageManager struct {
}

type RegistryStorageTarget struct {
	K8sConfig      string `json:"k8s_config"`
	K8sNamespace   string `json:"k8s_namespace"`
	DeploymentName string `json:"deployment_name"`
	ConfigMapName  string `json:"configmap_name"`
	CronJobName    string `json:"cronjob_name"`
}

type RegistryFilesystemStorageConfig struct {
	RootDirectory string `json:"root_directory"`
}

type RegistryS3StorageConfig struct {
	AccessKey      string `json:"s3_access_key"`
	SecretKey      string `json:"s3_secret_key"`
	Bucket         string `json:"s3_bucket"`
	Region         string `json:"s3_region"`
	RegionEndpoint string `json:"s3_endpoint"`
	RootDirectory  string `json:"s3_root_directory"`
	Secure         bool   `json:"s3_secure"`
	V4Auth         bool   `json:"v4auth"`
	Encrypt        bool   `json:"encrypt"`
	SkipVerify     bool   `json:"skip_verify"`
}

type RegistryStorageConfig struct {
	StorageType string                          `json:"storage_type"`
	Filesystem  RegistryFilesystemStorageConfig `json:"filesystem"`
	S3          RegistryS3StorageConfig         `json:"s3"`
	Resources   RegistryStorageResolvedResource `json:"resources"`
}

type RegistryStorageUpdateInput struct {
	RegistryStorageTarget
	StorageType string                  `json:"storage_type"`
	S3          RegistryS3StorageConfig `json:"s3"`
}

type RegistryStorageResolvedResource struct {
	K8sNamespace   string `json:"k8s_namespace"`
	DeploymentName string `json:"deployment_name"`
	ConfigMapName  string `json:"configmap_name"`
	CronJobName    string `json:"cronjob_name"`
}

func NewRegistryStorageManager() *RegistryStorageManager {
	return &RegistryStorageManager{}
}

func (m *RegistryStorageManager) TestS3Connection(config RegistryS3StorageConfig) error {
	if err := validateRegistryStorageUpdateInput(RegistryStorageUpdateInput{
		StorageType: RegistryStorageTypeS3,
		S3:          config,
	}); err != nil {
		return err
	}
	return validateRegistryS3Connection(context.Background(), config)
}

func (m *RegistryStorageManager) GetConfig(target RegistryStorageTarget) (*RegistryStorageConfig, error) {
	clientSet, resolved, err := m.resolveClientAndResources(target)
	slog.Info("cluster info", "info", resolved, "err", err)
	if err != nil {
		return nil, err
	}

	configMap, err := clientSet.CoreV1().ConfigMaps(resolved.K8sNamespace).Get(context.Background(), resolved.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(configMap.Data["registry_config.yaml"])
	if content == "" {
		return nil, errors.New("registry configmap 中 registry_config.yaml 为空")
	}

	storageType, filesystemCfg, s3Cfg, err := parseRegistryStorageConfig(content)
	slog.Info("storage info", "info", storageType, "filesystemCfg", filesystemCfg, "s3Cfg", s3Cfg, "err", err)
	if err != nil {
		return nil, err
	}

	return &RegistryStorageConfig{
		StorageType: storageType,
		Filesystem:  filesystemCfg,
		S3:          s3Cfg,
		Resources:   resolved,
	}, nil
}

func (m *RegistryStorageManager) UpdateConfig(input RegistryStorageUpdateInput) error {
	if err := validateRegistryStorageUpdateInput(input); err != nil {
		return err
	}

	clientSet, resolved, err := m.resolveClientAndResources(input.RegistryStorageTarget)
	slog.Info("cluster info", "info", resolved, "err", err)
	if err != nil {
		return err
	}

	ctx := context.Background()
	configMap, err := clientSet.CoreV1().ConfigMaps(resolved.K8sNamespace).Get(ctx, resolved.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	currentContent := strings.TrimSpace(configMap.Data["registry_config.yaml"])
	currentStorageType, _, currentS3Config, err := parseRegistryStorageConfig(currentContent)
	slog.Info("storage info", "info", currentStorageType, "currentS3Config", currentS3Config, "err", err)
	if err != nil {
		return err
	}
	if err = validateRegistryStorageTransition(currentStorageType, input.StorageType); err != nil {
		return err
	}
	if currentStorageType == RegistryStorageTypeS3 && input.StorageType == RegistryStorageTypeS3 {
		if isSameRegistryS3Config(currentS3Config, input.S3) {
			return errors.New("提交的 S3 配置与当前 registry 配置一致，无需更新")
		}
	}
	if input.StorageType == RegistryStorageTypeS3 {
		if err = validateRegistryS3Connection(ctx, input.S3); err != nil {
			return err
		}
	}

	nextContent, err := buildRegistryStorageConfig(configMap.Data["registry_config.yaml"], input)
	slog.Info("storage new info", "info", nextContent, "err", err)
	if err != nil {
		return err
	}
	configMap.Data["registry_config.yaml"] = nextContent
	if _, err = clientSet.CoreV1().ConfigMaps(resolved.K8sNamespace).Update(ctx, configMap, metav1.UpdateOptions{}); err != nil {
		return err
	}

	deployment, err := clientSet.AppsV1().Deployments(resolved.K8sNamespace).Get(ctx, resolved.DeploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	setRolloutAnnotation(&deployment.Spec.Template.ObjectMeta)
	if _, err = clientSet.AppsV1().Deployments(resolved.K8sNamespace).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return err
	}

	cronJob, err := clientSet.BatchV1().CronJobs(resolved.K8sNamespace).Get(ctx, resolved.CronJobName, metav1.GetOptions{})
	if err == nil {
		setRolloutAnnotation(&cronJob.Spec.JobTemplate.Spec.Template.ObjectMeta)
		if _, err = clientSet.BatchV1().CronJobs(resolved.K8sNamespace).Update(ctx, cronJob, metav1.UpdateOptions{}); err != nil {
			return err
		}
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

func validateRegistryStorageUpdateInput(input RegistryStorageUpdateInput) error {
	switch input.StorageType {
	case RegistryStorageTypeFilesystem:
		return nil
	case RegistryStorageTypeS3:
		if strings.TrimSpace(input.S3.AccessKey) == "" {
			return errors.New("s3.access_key 不能为空")
		}
		if strings.TrimSpace(input.S3.SecretKey) == "" {
			return errors.New("s3.secret_key 不能为空")
		}
		if strings.TrimSpace(input.S3.Bucket) == "" {
			return errors.New("s3.bucket 不能为空")
		}
		if strings.TrimSpace(input.S3.Region) == "" {
			return errors.New("s3.region 不能为空")
		}
		return nil
	default:
		return errors.New("storage_type 仅支持 filesystem 或 s3")
	}
}

func validateRegistryStorageTransition(currentStorageType string, targetStorageType string) error {
	if currentStorageType == RegistryStorageTypeS3 && targetStorageType == RegistryStorageTypeFilesystem {
		return errors.New("registry 存储只允许从 filesystem 切换到 s3，不支持从 s3 切回 filesystem")
	}
	return nil
}

func isSameRegistryS3Config(current RegistryS3StorageConfig, target RegistryS3StorageConfig) bool {
	return normalizeRegistryS3Config(current) == normalizeRegistryS3Config(target)
}

func normalizeRegistryS3Config(config RegistryS3StorageConfig) RegistryS3StorageConfig {
	config.AccessKey = strings.TrimSpace(config.AccessKey)
	config.SecretKey = strings.TrimSpace(config.SecretKey)
	config.Bucket = strings.TrimSpace(config.Bucket)
	config.Region = strings.TrimSpace(config.Region)
	config.RegionEndpoint = strings.TrimSpace(config.RegionEndpoint)
	config.RootDirectory = strings.TrimSpace(config.RootDirectory)
	return config
}

func (m *RegistryStorageManager) resolveClientAndResources(target RegistryStorageTarget) (*kubernetes.Clientset, RegistryStorageResolvedResource, error) {
	clientSet, err := m.GetK8sClient(target.K8sConfig)
	if err != nil {
		return nil, RegistryStorageResolvedResource{}, err
	}

	return clientSet, RegistryStorageResolvedResource{
		K8sNamespace:   strings.TrimSpace(target.K8sNamespace),
		DeploymentName: strings.TrimSpace(target.DeploymentName),
		ConfigMapName:  strings.TrimSpace(target.ConfigMapName),
		CronJobName:    strings.TrimSpace(target.CronJobName),
	}, nil
}

func (m *RegistryStorageManager) GetK8sClient(k8sConfig string) (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error
	if k8sConfig != "" {
		config, err = clientcmd.RESTConfigFromKubeConfig([]byte(k8sConfig))
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	return kubernetes.NewForConfig(config)
}

func parseRegistryStorageConfig(content string) (string, RegistryFilesystemStorageConfig, RegistryS3StorageConfig, error) {
	cfg := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return "", RegistryFilesystemStorageConfig{}, RegistryS3StorageConfig{}, err
	}

	storage := toStringMap(cfg["storage"])
	if len(storage) == 0 {
		return "", RegistryFilesystemStorageConfig{}, RegistryS3StorageConfig{}, errors.New("registry_config.yaml 缺少 storage 配置")
	}

	if s3Raw, ok := storage["s3"]; ok {
		s3Map := toStringMap(s3Raw)
		return RegistryStorageTypeS3,
			RegistryFilesystemStorageConfig{RootDirectory: registryStorageMountPath},
			RegistryS3StorageConfig{
				AccessKey:      toString(s3Map["accesskey"]),
				SecretKey:      toString(s3Map["secretkey"]),
				Bucket:         toString(s3Map["bucket"]),
				Region:         toString(s3Map["region"]),
				RegionEndpoint: toString(s3Map["regionendpoint"]),
				RootDirectory:  toString(s3Map["rootdirectory"]),
				Secure:         toBool(s3Map["secure"]),
				V4Auth:         toBool(s3Map["v4auth"]),
				Encrypt:        toBool(s3Map["encrypt"]),
				SkipVerify:     toBool(s3Map["skipverify"]),
			},
			nil
	}

	filesystemMap := toStringMap(storage["filesystem"])
	return RegistryStorageTypeFilesystem,
		RegistryFilesystemStorageConfig{
			RootDirectory: firstNonEmpty(toString(filesystemMap["rootdirectory"]), registryStorageMountPath),
		},
		RegistryS3StorageConfig{},
		nil
}

func buildRegistryStorageConfig(content string, input RegistryStorageUpdateInput) (string, error) {
	cfg := map[string]interface{}{}
	if strings.TrimSpace(content) != "" {
		if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
			return "", err
		}
	}
	storage := toStringMap(cfg["storage"])
	if len(storage) == 0 {
		storage = map[string]interface{}{}
	}

	delete(storage, "filesystem")
	delete(storage, "s3")

	switch input.StorageType {
	case RegistryStorageTypeFilesystem:
		storage["filesystem"] = map[string]interface{}{
			"rootdirectory": registryStorageMountPath,
		}
	case RegistryStorageTypeS3:
		s3Cfg := map[string]interface{}{
			"accesskey":  strings.TrimSpace(input.S3.AccessKey),
			"secretkey":  strings.TrimSpace(input.S3.SecretKey),
			"bucket":     strings.TrimSpace(input.S3.Bucket),
			"region":     strings.TrimSpace(input.S3.Region),
			"secure":     input.S3.Secure,
			"v4auth":     input.S3.V4Auth,
			"encrypt":    input.S3.Encrypt,
			"skipverify": input.S3.SkipVerify,
		}
		if endpoint := strings.TrimSpace(input.S3.RegionEndpoint); endpoint != "" {
			s3Cfg["regionendpoint"] = endpoint
		}
		if rootDirectory := strings.TrimSpace(input.S3.RootDirectory); rootDirectory != "" {
			s3Cfg["rootdirectory"] = rootDirectory
		}
		storage["s3"] = s3Cfg
	}

	cfg["storage"] = storage
	yamlBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}

func setRolloutAnnotation(meta *metav1.ObjectMeta) {
	if meta.Annotations == nil {
		meta.Annotations = map[string]string{}
	}
	restartedAt := time.Now().UTC().Format(time.RFC3339Nano)
	meta.Annotations["registry.w7.cc/storage-updated-at"] = restartedAt
	meta.Annotations["kubectl.kubernetes.io/restartedAt"] = restartedAt
}

func toStringMap(input interface{}) map[string]interface{} {
	if input == nil {
		return map[string]interface{}{}
	}
	if result, ok := input.(map[string]interface{}); ok {
		return result
	}
	return map[string]interface{}{}
}

func toString(input interface{}) string {
	if input == nil {
		return ""
	}
	if value, ok := input.(string); ok {
		return value
	}
	return fmt.Sprintf("%v", input)
}

func toBool(input interface{}) bool {
	switch value := input.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true")
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type registryS3Cred struct {
	accessKey string
	secretKey string
}

func (c registryS3Cred) Retrieve(context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     c.accessKey,
		SecretAccessKey: c.secretKey,
	}, nil
}

func validateRegistryS3Connection(ctx context.Context, config RegistryS3StorageConfig) error {
	client, err := newRegistryS3ProbeClient(config)
	if err != nil {
		return err
	}

	bucket := strings.TrimSpace(config.Bucket)
	if _, err = client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		return fmt.Errorf("检测 S3 bucket 失败: %w", err)
	}

	return nil
}

func newRegistryS3ProbeClient(config RegistryS3StorageConfig) (*s3.Client, error) {
	s3Config := normalizeRegistryS3Config(config)

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = s3Config.SkipVerify

	awsConfig := aws.Config{
		Region: s3Config.Region,
		Credentials: registryS3Cred{
			accessKey: s3Config.AccessKey,
			secretKey: s3Config.SecretKey,
		},
		HTTPClient: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}

	return s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		if s3Config.RegionEndpoint != "" {
			options.BaseEndpoint = aws.String(s3Config.RegionEndpoint)
			options.UsePathStyle = true
		}
	}), nil
}
