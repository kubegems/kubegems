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
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"kubegems.io/kubegems/pkg/log"
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
