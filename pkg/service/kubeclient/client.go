package kubeclient

import (
	"context"

	"kubegems.io/pkg/utils/agents"
)

type KubeClient struct {
	agentsClientSet *agents.ClientSet
}

var _kubeClient = &KubeClient{}

func Init(agents *agents.ClientSet) *KubeClient {
	_kubeClient = &KubeClient{agentsClientSet: agents}
	return _kubeClient
}

// Deprecated: 将依赖内置到调用方内部，避免使用全局单例
func GetClient() *KubeClient {
	return _kubeClient
}

// 获取集群的 代理客户端
func (k KubeClient) GetAgentClient(clusterName string) (*agents.HttpClient, error) {
	cli, err := k.agentsClientSet.ClientOf(context.TODO(), clusterName)
	if err != nil {
		return nil, err
	}
	_ = cli
	return nil, nil
}
