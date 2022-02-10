package kubeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/kubegems/gems/pkg/handlers"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/utils/agents"
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
func DoRequest(method, cluster, url string, body interface{}, into interface{}) error {
	return _kubeClient.DoRequest(method, cluster, url, body, into)
}

// Deprecated: 将依赖内置到调用方内部，避免使用全局单例
func GetTypedClient(ctx context.Context, cluster string) (*agents.TypedClient, error) {
	return _kubeClient.GetTypedClient(ctx, cluster)
}

// Deprecated: 将依赖内置到调用方内部，避免使用全局单例
func GetClient() *KubeClient {
	return _kubeClient
}

// Deprecated: 将依赖内置到调用方内部，避免使用全局单例
func Execute(ctx context.Context, cluster string, fn func(*agents.TypedClient) error) error {
	tc, err := _kubeClient.GetTypedClient(ctx, cluster)
	if err != nil {
		return err
	}
	if err := fn(tc); err != nil {
		log.Error(err, "k8s execute failed")
		return err
	}
	return nil
}

func (k KubeClient) GetTypedClient(ctx context.Context, cluster string) (*agents.TypedClient, error) {
	cli, err := k.agentsClientSet.ClientOf(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cli.TypedClient, nil
}

// 获取集群的 代理客户端
func (k KubeClient) GetAgentClient(clusterName string) (*agents.HttpClient, error) {
	cli, err := k.agentsClientSet.ClientOf(context.TODO(), clusterName)
	if err != nil {
		return nil, err
	}
	return cli.HttpClient, nil
}

func (k KubeClient) DoRequest(method, cluster, url string, body interface{}, into interface{}) error {
	return k.request(method, cluster, url, body, into)
}

func (k KubeClient) request(method, cluster, path string, body interface{}, into interface{}) error {
	agentClient, err := k.GetAgentClient(cluster)
	if err != nil {
		return err
	}
	target := agentClient.BaseAddr + path
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(method, target, bytes.NewReader(b))
	if err != nil {
		return err
	}

	resp, err := agentClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	response := &handlers.ResponseStruct{Data: into}
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(response.Message)
	}
	return nil
}

func formatURL(args, labelsel, query map[string]string, ptn string) string {
	base := ptn
	for key, value := range args {
		base = strings.ReplaceAll(base, "{"+key+"}", value)
	}
	qs := url.Values{}
	for qk, qv := range labelsel {
		qs.Set("labels["+qk+"]", qv)
	}
	for qk, qv := range query {
		qs.Set(qk, qv)
	}
	u := url.URL{
		Path:     base,
		RawQuery: qs.Encode(),
	}
	return u.String()
}
