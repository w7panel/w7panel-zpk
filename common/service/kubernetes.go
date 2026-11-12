package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Kubernetes struct{ Client kubernetes.Interface }

func (s Kubernetes) NewClient(config *rest.Config) (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(config)
}

func (s Kubernetes) Namespaces(ctx context.Context) ([]string, error) {
	list, err := s.Client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		names = append(names, item.Name)
	}
	return names, nil
}

func (s Kubernetes) Applications(ctx context.Context, namespace, kind string) ([]string, error) {
	names := []string{}
	switch kind {
	case "deployments":
		list, err := s.Client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "daemonsets":
		list, err := s.Client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "statefulsets":
		list, err := s.Client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	}
	return names, nil
}

func (s Kubernetes) Containers(ctx context.Context, namespace, kind, name string) ([]string, error) {
	containers := []corev1.Container{}
	switch kind {
	case "deployments":
		item, err := s.Client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		containers = item.Spec.Template.Spec.Containers
	case "daemonsets":
		item, err := s.Client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		containers = item.Spec.Template.Spec.Containers
	case "statefulsets":
		item, err := s.Client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		containers = item.Spec.Template.Spec.Containers
	}
	names := make([]string, 0, len(containers))
	for _, item := range containers {
		names = append(names, item.Name)
	}
	return names, nil
}

func (s Kubernetes) PatchImage(ctx context.Context, namespace, kind, name, container, image string) (string, error) {
	patch, err := json.Marshal(map[string]interface{}{"spec": map[string]interface{}{"template": map[string]interface{}{"spec": map[string]interface{}{"containers": []map[string]string{{"name": container, "image": image}}}}}})
	if err != nil {
		return "", err
	}
	var status interface{}
	switch kind {
	case "deployments":
		item, e := s.Client.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
		if e != nil {
			return "", e
		}
		status = item.Status
	case "daemonsets":
		item, e := s.Client.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
		if e != nil {
			return "", e
		}
		status = item.Status
	case "statefulsets":
		item, e := s.Client.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
		if e != nil {
			return "", e
		}
		status = item.Status
	default:
		return "", errors.New("部署控制器类型无效")
	}
	content, err := json.Marshal(status)
	return string(content), err
}

func (s Kubernetes) Proxy(config *rest.Config, writer http.ResponseWriter, request *http.Request, path string) error {
	target, err := url.Parse(config.Host)
	if err != nil {
		return err
	}
	transport, err := rest.TransportFor(config)
	if err != nil {
		return err
	}
	request.URL.Path = path
	request.Header.Set("Authorization", "Bearer "+config.BearerToken)
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	proxy.ServeHTTP(writer, request)
	return nil
}
