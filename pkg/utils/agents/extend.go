package agents

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/alertmanager/pkg/labels"
	alertmanagertypes "github.com/prometheus/alertmanager/types"
	"kubegems.io/pkg/apis/plugins"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/prometheus"
)

var (
	alertProxyHeader = map[string]string{
		"namespace": "gemcloud-monitoring-system",
		"service":   "alertmanager",
		"port":      "9093",
	}
	silenceCommentPrefix = "fingerprint-"
)

type ExtendClient struct {
	*TypedClient
}

// plugins.kubegems.io/v1alpha1
func (c *ExtendClient) ListPlugins(ctx context.Context) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	err := c.DoRequest(ctx, Request{
		Method: http.MethodGet,
		Path:   "/custom/" + plugins.GroupName + "/v1beta1/installers",
		Into:   WrappedResponse(&ret),
	})
	return ret, err
}

func (c *ExtendClient) EnablePlugin(ctx context.Context, ptype, name string) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/custom/%s/v1beta1/installers/%s/actions/enable?type=%s", plugins.GroupName, name, ptype),
	})
}

func (c *ExtendClient) DisablePlugin(ctx context.Context, ptype, name string) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPut,
		Path:   fmt.Sprintf("/custom/%s/v1beta1/installers/%s/actions/disable?type=%s", plugins.GroupName, name, ptype),
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

func (c *ExtendClient) ListSilences(ctx context.Context, labels map[string]string) ([]alertmanagertypes.Silence, error) {
	allSilences := []alertmanagertypes.Silence{}

	req := Request{
		Path: "/v1/service-proxy/api/v2/silences",
		Query: func() url.Values {
			values := url.Values{}
			for k, v := range labels {
				values.Add("filter", fmt.Sprintf(`%s="%s"`, k, v))
			}
			return values
		}(),
		Headers: HeadersFrom(alertProxyHeader),
		Into:    &allSilences,
	}

	if err := c.DoRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("list silence by %v, %w", labels, err)
	}
	// 只返回活跃的
	ret := []alertmanagertypes.Silence{}
	for _, v := range allSilences {
		if v.Status.State == alertmanagertypes.SilenceStateActive &&
			strings.HasPrefix(v.Comment, silenceCommentPrefix) {
			ret = append(ret, v)
		}
	}
	return ret, nil
}

func (c *ExtendClient) CreateOrUpdateSilenceIfNotExist(ctx context.Context, info models.AlertInfo) error {
	silenceList, err := c.ListSilences(ctx, info.LabelMap)
	if err != nil {
		return err
	}
	silence := convertBlackListToSilence(info)
	switch len(silenceList) {
	case 0:
		break
	case 1:
		silence.ID = silenceList[0].ID
	default:
		return fmt.Errorf("too many silences for alert: %v", info)
	}

	agentreq := Request{
		Method:  http.MethodPost,
		Path:    "/v1/service-proxy/api/v2/silences",
		Body:    silence,
		Headers: HeadersFrom(alertProxyHeader),
	}

	if err := c.DoRequest(ctx, agentreq); err != nil {
		return fmt.Errorf("create silence:%w", err)
	}
	return nil
}

func (c *ExtendClient) DeleteSilenceIfExist(ctx context.Context, info models.AlertInfo) error {
	silenceList, err := c.ListSilences(ctx, info.LabelMap)
	if err != nil {
		return err
	}
	switch len(silenceList) {
	case 0:
		return nil
	case 1:
		agentreq := Request{
			Method:  http.MethodDelete,
			Path:    fmt.Sprintf("/v1/service-proxy/api/v2/silences/%s", silenceList[0].ID),
			Headers: HeadersFrom(alertProxyHeader),
		}
		return c.DoRequest(ctx, agentreq)
	default:
		return fmt.Errorf("too many silences for alert: %v", info)
	}
}

func (c *ExtendClient) CheckAlertmanagerConfig(ctx context.Context, data *v1alpha1.AlertmanagerConfig) error {
	return c.DoRequest(ctx, Request{
		Method: http.MethodPost,
		Path:   "/custom/alertmanager/v1/alerts/_/actions/check",
		Body:   data,
	})
}

// TODO: 使用原生prometheus api
func (c *ExtendClient) GetPromeAlertRules(ctx context.Context, name string) (map[string]prometheus.RealTimeAlertRule, error) {
	ret := map[string]prometheus.RealTimeAlertRule{}
	if err := c.DoRequest(ctx, Request{
		Path: fmt.Sprintf("/custom/prometheus/v1/alertrule?name=%s", name),
		Into: WrappedResponse(&ret),
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func convertBlackListToSilence(info models.AlertInfo) alertmanagertypes.Silence {
	ret := alertmanagertypes.Silence{
		StartsAt:  *info.SilenceStartsAt,
		EndsAt:    *info.SilenceEndsAt,
		UpdatedAt: *info.SilenceUpdatedAt,
		CreatedBy: info.SilenceCreator,
		Comment:   fmt.Sprintf("%s%s", silenceCommentPrefix, info.Fingerprint), // comment存指纹，以便取出时做匹配
		Matchers:  make(labels.Matchers, len(info.LabelMap)),
	}
	index := 0
	for k, v := range info.LabelMap {
		ret.Matchers[index] = &labels.Matcher{
			Type:  labels.MatchEqual,
			Name:  k,
			Value: v,
		}
		index++
	}
	return ret
}
