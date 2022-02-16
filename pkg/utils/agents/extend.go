package agents

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"kubegems.io/pkg/apis/plugins"
)

type ExtendClient struct {
	*TypedClient
}

// plugins.kubegems.io/v1alpha1
func (c *ExtendClient) ListPlugins(ctx context.Context) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	err := c.DoRequest(ctx, Request{
		Method: http.MethodGet,
		Path:   "/custom/" + plugins.GroupName + "/v1alpha1/plugins",
		Into:   WrappedResponse(ret),
	})
	return ret, err
}

func (c *ExtendClient) EnablePlugin(ctx context.Context, ptype, name string) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPost,
		Path:   fmt.Sprintf("/custom/%s/v1alpha1/plugins/%s/actions/enable?type=%s", plugins.GroupName, name, ptype),
	})
}

func (c *ExtendClient) DisablePlugin(ctx context.Context, ptype, name string) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodDelete,
		Path:   fmt.Sprintf("/custom/%s/v1alpha1/plugins/%s/actions/disable?type=%s", plugins.GroupName, name, ptype),
	})
}

// statistics.system/v1
func (c *ExtendClient) ClusterWorkloadStatistics(ctx context.Context, ret interface{}) error {
	return c.DoRequest(ctx, Request{
		Path: "/custom/statistics.system/v1/workloads",
		Into: WrappedResponse(ret),
	})
}

func (c *ExtendClient) ClusterResourceStatistics(ctx context.Context, ret interface{}) error {
	return c.DoRequest(ctx, Request{
		Path: "/custom/statistics.system/v1/resources",
		Into: WrappedResponse(ret),
	})
}

// health.system/v1
func (c *ExtendClient) Healthy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.DoRequest(ctx, Request{Path: "/healthz"})
}
