package logic

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	K8sDeployWhenTagUpdate = int32(1)
	K8sDeployWhenTagAdd    = int32(2)
)

type Deploy struct{}

func (l Deploy) MakeK8sConfig(k8sConfig string) (*rest.Config, error) {
	k8sConfig, err := l.DecryptK8sConfig(k8sConfig)
	if err != nil {
		return nil, err
	}
	var config *rest.Config
	if k8sConfig != "" {
		if err := l.ValidateK8sConfig(k8sConfig); err != nil {
			return nil, err
		}
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

	return config, nil
}

func (l Deploy) GetK8sClient(k8sConfig string) (*kubernetes.Clientset, error) {
	config, err := l.MakeK8sConfig(k8sConfig)
	if err != nil {
		return nil, err
	}

	return (service.Kubernetes{}).NewClient(config)
}

const encryptedK8sConfigPrefix = "aes:"

// EncryptK8sConfig leaves the empty in-cluster configuration unchanged.
func (l Deploy) EncryptK8sConfig(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	// Normalize an encrypted input before saving to avoid double encryption.
	plain, err := l.DecryptK8sConfig(content)
	if err != nil {
		return "", err
	}
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	encrypted, err := function.AesEncrypt(plain, key)
	if err != nil {
		return "", errors.New("加密部署规则 KUBECONFIG 失败")
	}
	return encryptedK8sConfigPrefix + encrypted, nil
}

// DecryptK8sConfig accepts historical plaintext; marked ciphertext must decrypt successfully.
func (l Deploy) DecryptK8sConfig(content string) (string, error) {
	if !strings.HasPrefix(content, encryptedK8sConfigPrefix) {
		return content, nil
	}
	key := function.GetMd5(facade.GetConfig().GetString("setting.secret"))
	plain, err := function.AesDecrypt(strings.TrimPrefix(content, encryptedK8sConfigPrefix), key)
	if err != nil || plain == "" {
		return "", errors.New("解密部署规则 KUBECONFIG 失败，请检查 setting.secret 或重新保存配置")
	}
	return plain, nil
}

// ValidateK8sConfig keeps credential loading self-contained inside task Secrets.
func (l Deploy) ValidateK8sConfig(content string) error {
	cfg, err := clientcmd.Load([]byte(content))
	if err != nil || clientcmd.Validate(*cfg) != nil {
		return errors.New("KUBECONFIG 无效")
	}
	for _, cluster := range cfg.Clusters {
		if cluster.CertificateAuthority != "" {
			return errors.New("请使用内嵌 CA 数据的 KUBECONFIG")
		}
	}
	for _, auth := range cfg.AuthInfos {
		if auth.Exec != nil || auth.AuthProvider != nil || auth.TokenFile != "" || auth.ClientCertificate != "" || auth.ClientKey != "" {
			return errors.New("KUBECONFIG 不支持外部认证程序或本地凭证文件")
		}
	}
	return nil
}

func (l Deploy) Namespaces(ctx context.Context, kubeconfig string) ([]string, error) {
	client, err := l.GetK8sClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return (service.Kubernetes{Client: client}).Namespaces(ctx)
}

func (l Deploy) Applications(ctx context.Context, kubeconfig, namespace, controllerType string) ([]string, error) {
	client, err := l.GetK8sClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return (service.Kubernetes{Client: client}).Applications(ctx, namespace, controllerType)
}

func (l Deploy) Containers(ctx context.Context, kubeconfig, namespace, controllerType, appName string) ([]string, error) {
	client, err := l.GetK8sClient(kubeconfig)
	if err != nil {
		return nil, err
	}
	return (service.Kubernetes{Client: client}).Containers(ctx, namespace, controllerType, appName)
}

func (l Deploy) Proxy(writer http.ResponseWriter, request *http.Request, path string) error {
	config, err := l.MakeK8sConfig("")
	if err != nil {
		return err
	}
	return (service.Kubernetes{}).Proxy(config, writer, request, path)
}
