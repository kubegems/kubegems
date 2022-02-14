package kubeclient

import (
	"fmt"
	"net/http"

	"kubegems.io/pkg/apis/plugins"
)

func (k KubeClient) ListPlugins(cluster string) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	url := fmt.Sprintf("/custom/%s/v1alpha1/plugins", plugins.GroupName)
	err := k.DoRequest(http.MethodGet, cluster, url, nil, &ret)
	return ret, err
}

func (k KubeClient) EnablePlugin(cluster, plugintype, plugin string) error {
	url := fmt.Sprintf("/custom/%s/v1alpha1/plugins/%s/actions/enable?type=%s", plugins.GroupName, plugin, plugintype)
	return k.DoRequest(http.MethodPost, cluster, url, nil, nil)
}

func (k KubeClient) DisablePlugin(cluster, plugintype, plugin string) error {
	url := fmt.Sprintf("/custom/%s/v1alpha1/plugins/%s/actions/disable?type=%s", plugins.GroupName, plugin, plugintype)
	return k.DoRequest(http.MethodPost, cluster, url, nil, nil)
}
