// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubegems.io/kubegems/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func GetKubeRestConfig(kubeconfig []byte) (*rest.Config, error) {
	return clientcmd.RESTConfigFromKubeConfig(kubeconfig)
}

func NewLocalClient() (client.WithWatch, error) {
	cfg, err := AutoClientConfig()
	if err != nil {
		return nil, err
	}
	return client.NewWithWatch(cfg, client.Options{})
}

// AutoClientConfig 自动获取当前环境 restConfig
func AutoClientConfig() (*rest.Config, error) {
	return DefaultClientConfig().ClientConfig()
}

// DefaultClientConfig read config from kubeconfig or incluster config as fallback
// It read config from KUBECONFIG environment file or default kubeconfig file or in cluster config.
// https://github.com/kubernetes/client-go/blob/cab7ba1d4a523956b6395dcbe38620159ac43fef/tools/clientcmd/loader.go#L143-L152
func DefaultClientConfig() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), nil)
}

// LocalNamespace return namespace of default namespace of the kubeconfig or current pod namespace
// Out of cluster: It read from kubeconfig default context namespace.
// On cluster: It read from POD_NAMESPACE environment or the serviceaccount namespace file.
// https://github.com/kubernetes/client-go/blob/cab7ba1d4a523956b6395dcbe38620159ac43fef/tools/clientcmd/client_config.go#L581-L596
func LocalNamespaceOrDefault(def string) string {
	ns, _, _ := DefaultClientConfig().Namespace()
	if ns == "" || ns == "default" {
		return def
	}
	return ns
}
