package kube

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubegems.io/pkg/log"
)

func GetKubeClient(kubeconfig []byte) (*rest.Config, *kubernetes.Clientset, error) {
	restconfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, nil, err
	}
	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return restconfig, nil, err
	}
	return restconfig, clientset, nil
}

func GetKubeconfigInfos(kubeconfig []byte) (apiserver string, cert, key, ca []byte, err error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		log.Errorf("unable decode kubeconfig:%v", err)
		return "", nil, nil, nil, err
	}
	return cfg.Host, cfg.TLSClientConfig.CertData, cfg.TLSClientConfig.KeyData, cfg.TLSClientConfig.CAData, nil
}

// AutoClientConfig 自动获取当前环境 restConfig
// 1. 先尝试使用InClusterConfig,
// 2. 若不存在则使用 ~/.kube/config，
// 3. 否不存在则失败
func AutoClientConfig() (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err != nil {
		home, _ := os.UserHomeDir()
		return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
	} else {
		return config, nil
	}
}
