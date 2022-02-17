package alerthandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus/alertmanager/pkg/labels"
	alerttypes "github.com/prometheus/alertmanager/types"
	"github.com/prometheus/common/model"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
)

const (
	allNamespace = "_all"
)

var (
	defaultReceiverName = "gemcloud-default-webhook"
	defaultReceiverURL  = "http://gems-agent.gemcloud-system:8041/alert"
	defaultReceiver     = v1alpha1.Receiver{
		Name: defaultReceiverName,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL: &defaultReceiverURL,
			},
		},
	}

	nullReceiverName = "null"
	nullReceiver     = v1alpha1.Receiver{Name: nullReceiverName}
)

func (h *AlertsHandler) getRawAlertResource(ctx context.Context, cluster, namespace string) (*prometheus.RawAlertResource, error) {
	amcfg, err := h.getOrCreateAlertmanagerConfig(ctx, cluster, namespace)
	if err != nil {
		return nil, err
	}

	promerule, err := h.getOrCreatePrometheusRule(ctx, cluster, namespace)
	if err != nil {
		return nil, err
	}

	silence, err := h.ListSilences(ctx, cluster, namespace)
	if err != nil {
		return nil, err
	}

	return &prometheus.RawAlertResource{
		AlertmanagerConfig: amcfg,
		PrometheusRule:     promerule,
		Silences:           silence,
	}, nil
}

func (h *AlertsHandler) ListSilences(ctx context.Context, cluster string, namespace string) ([]*alerttypes.Silence, error) {
	silences := []*alerttypes.Silence{}

	err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		url := "/custom/alertmanager/v1/silence"
		if namespace != "" {
			url = fmt.Sprintf(`%s?filter=%s="%s"`, url, prometheus.AlertNamespaceLabel, namespace)
		}
		return cli.DoRequest(ctx, agents.Request{
			Path: url,
			Into: agents.WrappedResponse(&silences),
		})
	})
	if err != nil {
		return nil, err
	}

	// 只返回活跃的
	var ret []*alerttypes.Silence
	for _, v := range silences {
		if v.Status.State == alerttypes.SilenceStateActive {
			ret = append(ret, v)
		}
	}

	return ret, nil
}

func (h *AlertsHandler) GetSilence(ctx context.Context, cluster, namespace, alertName string) (*alerttypes.Silence, error) {
	silences := []*alerttypes.Silence{}

	err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		// 只有一个filter生效
		url := "/custom/alertmanager/v1/silence"
		if namespace != "" {
			url = fmt.Sprintf(`%s?%s="%s"`, url, prometheus.AlertNamespaceLabel, namespace)
		}
		return cli.DoRequest(ctx, agents.Request{
			Path: url,
			Into: agents.WrappedResponse(&silences),
		})
	})
	if err != nil {
		return nil, err
	}

	// 只返回活跃的
	actives := []*alerttypes.Silence{}
	for _, silence := range silences {
		if silence.Status.State == alerttypes.SilenceStateActive &&
			silence.Matchers.Matches(model.LabelSet{
				prometheus.AlertNamespaceLabel: model.LabelValue(namespace),
				prometheus.AlertNameLabel:      model.LabelValue(alertName),
			}) { // 名称匹配
			actives = append(actives, silence)
		}
	}
	if len(actives) == 0 {
		return nil, nil
	}
	if len(actives) > 1 {
		return nil, errors.New("too many silences")
	}

	return actives[0], nil
}

func (h *AlertsHandler) CreateSilenceIfNotExist(ctx context.Context, cluster, namespace, alertName string) error {
	silence, err := h.GetSilence(ctx, cluster, namespace, alertName)
	if err != nil {
		return err
	}
	// 不存在，创建
	if silence == nil {
		silence = &alerttypes.Silence{
			Comment:   fmt.Sprintf("silence for %s", alertName),
			CreatedBy: alertName,
			Matchers: labels.Matchers{
				&labels.Matcher{
					Type:  labels.MatchEqual,
					Name:  prometheus.AlertNamespaceLabel,
					Value: namespace,
				},
				&labels.Matcher{
					Type:  labels.MatchEqual,
					Name:  prometheus.AlertNameLabel,
					Value: alertName,
				},
			},
			StartsAt: time.Now(),
			EndsAt:   time.Now().AddDate(1000, 0, 0), // 100年
		}

		// create
		return h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
			return cli.DoRequest(ctx, agents.Request{
				Method: http.MethodPost,
				Path:   "/custom/alertmanager/v1/silence/_/actions/create",
				Body:   silence,
			})
		})
	}
	return nil
}

func (h *AlertsHandler) DeleteSilenceIfExist(ctx context.Context, cluster, namespace, alertName string) error {
	silence, err := h.GetSilence(ctx, cluster, namespace, alertName)
	if err != nil {
		return err
	}
	// 存在，删除
	if silence != nil {
		values := url.Values{}
		values.Add("id", silence.ID)

		return h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
			return cli.DoRequest(ctx, agents.Request{
				Method: http.MethodDelete,
				Path:   "/custom/alertmanager/v1/silence/_/actions/delete",
				Query:  values,
			})
		})
	}
	return nil
}

func (h *AlertsHandler) getOrCreateAlertmanagerConfig(ctx context.Context, cluster, namespace string) (*v1alpha1.AlertmanagerConfig, error) {
	aconfig := &v1alpha1.AlertmanagerConfig{}
	err := h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		err := tc.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.AlertmanagerConfigName}, aconfig)
		if kerrors.IsNotFound(err) {
			// 初始化
			aconfig = prometheus.GetBaseAlertmanagerConfig(namespace)
			if err := h.CheckAlertmanagerConfig(ctx, cluster, aconfig); err != nil {
				return err
			}

			return tc.Create(ctx, aconfig)
		} else {
			return err
		}
	})

	return aconfig, err
}

func (h *AlertsHandler) CheckAlertmanagerConfig(ctx context.Context, cluster string, data *v1alpha1.AlertmanagerConfig) error {
	return h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.DoRequest(ctx, agents.Request{
			Method: http.MethodPost,
			Path:   "/custom/alertmanager/v1/alerts/_/actions/check",
			Body:   data,
		})
	})
}

func (h *AlertsHandler) UpdateAlertmanagerConfig(ctx context.Context, cluster string, data *v1alpha1.AlertmanagerConfig) (*v1alpha1.AlertmanagerConfig, error) {
	if err := h.CheckAlertmanagerConfig(ctx, cluster, data); err != nil {
		log.Errorf("check alertmanager config failed: %v", err)
		return data, err
	}

	if err := h.Execute(ctx, cluster, func(ctx context.Context, cli agents.Client) error {
		return cli.Update(ctx, data)
	}); err != nil {
		return nil, err
	}
	return data, nil
}

func (h *AlertsHandler) getOrCreatePrometheusRule(ctx context.Context, cluster, namespace string) (*monitoringv1.PrometheusRule, error) {
	prule := &monitoringv1.PrometheusRule{}
	err := h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		err := tc.Get(ctx, types.NamespacedName{Namespace: namespace, Name: prometheus.PrometheusRuleName}, prule)
		if kerrors.IsNotFound(err) {
			prule = prometheus.GetBasePrometheusRule(namespace)
			return tc.Create(ctx, prule)
		} else {
			return err
		}
	})

	return prule, err
}

func (h *AlertsHandler) commitToK8s(ctx context.Context, cluster string, raw *prometheus.RawAlertResource) error {
	if err := h.CheckAlertmanagerConfig(ctx, cluster, raw.AlertmanagerConfig); err != nil {
		return err
	}

	return h.Execute(ctx, cluster, func(ctx context.Context, tc agents.Client) error {
		if err := tc.Update(ctx, raw.PrometheusRule); err != nil {
			return err
		}
		return tc.Update(ctx, raw.AlertmanagerConfig)
	})
}
