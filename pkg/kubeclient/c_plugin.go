package kubeclient

import (
	"fmt"
	"net/http"
)

func (k KubeClient) ListPlugins(cluster string) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	err := k.DoRequest(http.MethodGet, cluster, "/custom/plugins.gems.cloudminds.com/v1alpha1/plugins", nil, &ret)
	return ret, err
}

func (k KubeClient) EnablePlugin(cluster, plugintype, plugin string) error {
	url := fmt.Sprintf("/custom/plugins.gems.cloudminds.com/v1alpha1/plugins/%s/actions/enable?type=%s", plugin, plugintype)
	return k.DoRequest(http.MethodPost, cluster, url, nil, nil)
}

func (k KubeClient) DisablePlugin(cluster, plugintype, plugin string) error {
	url := fmt.Sprintf("/custom/plugins.gems.cloudminds.com/v1alpha1/plugins/%s/actions/disable?type=%s", plugin, plugintype)
	return k.DoRequest(http.MethodPost, cluster, url, nil, nil)
}
