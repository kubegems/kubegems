package apis

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/prometheus-operator/prometheus-operator/pkg/assets"
	"github.com/prometheus/alertmanager/client"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/client_golang/api"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// 获取各个集群的告警信息
type AlertmanagerHandler struct {
	client api.Client
	C      kubernetes.Interface
}

func NewAlertmanagerClient(server string, k8sclient kubernetes.Interface) (*AlertmanagerHandler, error) {
	client, err := api.NewClient(api.Config{Address: server})
	if err != nil {
		return nil, err
	}
	return &AlertmanagerHandler{
		client: client,
		C:      k8sclient,
	}, nil
}

type alertQuery struct {
	Filter      string `form:"filter"`
	Receiver    string `form:"receiver"`
	Silenced    bool   `form:"silenced"`
	Inhibited   bool   `form:"inbibited"`
	Active      bool   `form:"active"`
	Unporcessed bool   `form:"unprocessed"`
}

// @Tags         Agent.V1
// @Summary      检查alertmanagerconfig
// @Description  检查alertmanagerconfig
// @Accept       json
// @Produce      json
// @Param        form  body      v1alpha1.AlertmanagerConfig           true  "body"
// @Success      200   {object}  handlers.ResponseStruct{Data=string}  ""
// @Router       /v1/proxy/cluster/{cluster}/custom/alertmanager/v1/alerts/_/actions/check [post]
// @Security     JWT
func (h *AlertmanagerHandler) CheckConfig(c *gin.Context) {
	config := &v1alpha1.AlertmanagerConfig{}
	if err := c.BindJSON(config); err != nil {
		NotOK(c, err)
		return
	}
	if err := checkAlertmanagerConfig(c.Request.Context(), config, assets.NewStore(h.C.CoreV1(), h.C.CoreV1())); err != nil {
		NotOK(c, err)
		return
	}
	OK(c, "")
}

// @Tags         Agent.V1
// @Summary      获取alertmanager中的告警数据
// @Description  获取alertmanager中的告警数据
// @Accept       json
// @Produce      json
// @Param        cluster      path      string                                                true   "cluster"
// @Param        filter       query     string                                                false  "filter"
// @Param        receiver     query     string                                                false  "receiver"
// @Param        silenced     query     bool                                                  false  "silenced"
// @Param        inhibited    query     bool                                                  false  "inhibited"
// @Param        active       query     bool                                                  false  "active"
// @Param        unprocessed  query     bool                                                  false  "unprocessed"
// @Success      200          {object}  handlers.ResponseStruct{Data=[]client.ExtendedAlert}  "labelvalues"
// @Router       /v1/proxy/cluster/{cluster}/custom/alertmanager/v1/alerts [get]
// @Security     JWT
func (h *AlertmanagerHandler) ListAlerts(c *gin.Context) {
	alertapi := client.NewAlertAPI(h.client)
	query := &alertQuery{}
	_ = c.BindQuery(query)
	alerts, err := alertapi.List(c.Request.Context(), query.Filter, query.Receiver, query.Silenced, query.Inhibited, query.Active, query.Unporcessed)
	if err != nil {
		NotOK(c, err)
	}
	OK(c, alerts)
}

// @Tags         Agent.V1
// @Summary      为指定告警规则添加silence
// @Description  添加告警silence
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true  "cluster"
// @Param        from     body      types.Silence                         true  "silence"
// @Success      200      {object}  handlers.ResponseStruct{Data=string}  ""
// @Router       /v1/proxy/cluster/{cluster}/custom/alertmanager/v1/silence/_/actions/create [post]
// @Security     JWT
func (h *AlertmanagerHandler) CreateSilence(c *gin.Context) {
	silenceapi := client.NewSilenceAPI(h.client)
	silence := types.Silence{}
	c.BindJSON(&silence)
	if _, err := silenceapi.Set(c.Request.Context(), silence); err != nil {
		NotOK(c, err)
	}
	OK(c, "")
}

// @Tags         Agent.V1
// @Summary      get silence
// @Description  get silence
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true  "cluster"
// @Param        filter   query     string                                true  "filter"
// @Success      200      {object}  handlers.ResponseStruct{Data=string}  ""
// @Router       /v1/proxy/cluster/{cluster}/custom/alertmanager/v1/silence [get]
// @Security     JWT
func (h *AlertmanagerHandler) ListSilence(c *gin.Context) {
	silenceapi := client.NewSilenceAPI(h.client)
	ret, err := silenceapi.List(c.Request.Context(), c.Query("filter"))
	if err != nil {
		NotOK(c, err)
	}
	OK(c, ret)
}

// @Tags         Agent.V1
// @Summary      get silence
// @Description  get silence
// @Accept       json
// @Produce      json
// @Param        cluster  path      string                                true  "cluster"
// @Param        id       query     string                                true  "id"
// @Success      200      {object}  handlers.ResponseStruct{Data=string}  ""
// @Router       /v1/proxy/cluster/{cluster}/custom/alertmanager/v1/silence/_/actions/delete [delete]
// @Security     JWT
func (h *AlertmanagerHandler) DeleteSilence(c *gin.Context) {
	silenceapi := client.NewSilenceAPI(h.client)
	if err := silenceapi.Expire(c.Request.Context(), c.Query("id")); err != nil {
		NotOK(c, err)
	}
	OK(c, "")
}

// COPY from proemtheus operator  pkg/alertmanager/operator.go

// checkAlertmanagerConfig verifies that an AlertmanagerConfig object is valid
// and has no missing references to other objects.
func checkAlertmanagerConfig(ctx context.Context, amc *v1alpha1.AlertmanagerConfig, store *assets.Store) error {
	receiverNames, err := checkReceivers(ctx, amc, store)
	if err != nil {
		return err
	}

	return checkAlertmanagerRoutes(amc.Spec.Route, receiverNames, true)
}

func checkReceivers(ctx context.Context, amc *v1alpha1.AlertmanagerConfig, store *assets.Store) (map[string]struct{}, error) {
	var err error
	receiverNames := make(map[string]struct{})

	for i, receiver := range amc.Spec.Receivers {
		if _, found := receiverNames[receiver.Name]; found {
			return nil, errors.Errorf("%q receiver is not unique", receiver.Name)
		}
		receiverNames[receiver.Name] = struct{}{}

		amcKey := fmt.Sprintf("alertmanagerConfig/%s/%s/%d", amc.GetNamespace(), amc.GetName(), i)

		err = checkPagerDutyConfigs(ctx, receiver.PagerDutyConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}

		err = checkOpsGenieConfigs(ctx, receiver.OpsGenieConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}
		err = checkSlackConfigs(ctx, receiver.SlackConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}

		err = checkWebhookConfigs(ctx, receiver.WebhookConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}

		err = checkWechatConfigs(ctx, receiver.WeChatConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}

		err = checkEmailConfigs(ctx, receiver.EmailConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}

		err = checkVictorOpsConfigs(ctx, receiver.VictorOpsConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}

		err = checkPushoverConfigs(ctx, receiver.PushoverConfigs, amc.GetNamespace(), amcKey, store)
		if err != nil {
			return nil, err
		}
	}

	return receiverNames, nil
}

func checkPagerDutyConfigs(ctx context.Context, configs []v1alpha1.PagerDutyConfig, namespace string, key string, store *assets.Store) error {
	for i, config := range configs {
		pagerDutyConfigKey := fmt.Sprintf("%s/pagerduty/%d", key, i)

		if config.RoutingKey != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.RoutingKey); err != nil {
				return err
			}
		}

		if config.ServiceKey != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.ServiceKey); err != nil {
				return err
			}
		}

		if err := configureHTTPConfigInStore(ctx, config.HTTPConfig, namespace, pagerDutyConfigKey, store); err != nil {
			return err
		}
	}

	return nil
}

func checkOpsGenieConfigs(ctx context.Context, configs []v1alpha1.OpsGenieConfig, namespace string, key string, store *assets.Store) error {
	for i, config := range configs {
		opsgenieConfigKey := fmt.Sprintf("%s/opsgenie/%d", key, i)

		if config.APIKey != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.APIKey); err != nil {
				return err
			}
		}

		if err := config.Validate(); err != nil {
			return err
		}

		if err := configureHTTPConfigInStore(ctx, config.HTTPConfig, namespace, opsgenieConfigKey, store); err != nil {
			return err
		}
	}

	return nil
}

func checkSlackConfigs(ctx context.Context, configs []v1alpha1.SlackConfig, namespace string, key string, store *assets.Store) error {
	for i, config := range configs {
		slackConfigKey := fmt.Sprintf("%s/slack/%d", key, i)

		if config.APIURL != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.APIURL); err != nil {
				return err
			}
		}

		if err := config.Validate(); err != nil {
			return err
		}

		if err := configureHTTPConfigInStore(ctx, config.HTTPConfig, namespace, slackConfigKey, store); err != nil {
			return err
		}
	}

	return nil
}

func checkWebhookConfigs(ctx context.Context, configs []v1alpha1.WebhookConfig, namespace string, key string, store *assets.Store) error {
	for i, config := range configs {
		webhookConfigKey := fmt.Sprintf("%s/webhook/%d", key, i)

		if config.URL == nil && config.URLSecret == nil {
			return errors.New("one of url or urlSecret should be specified")
		}

		if config.URLSecret != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.URLSecret); err != nil {
				return err
			}
		}

		if err := configureHTTPConfigInStore(ctx, config.HTTPConfig, namespace, webhookConfigKey, store); err != nil {
			return err
		}
	}

	return nil
}

func checkWechatConfigs(ctx context.Context, configs []v1alpha1.WeChatConfig, namespace string, key string, store *assets.Store) error {
	for i, config := range configs {
		wechatConfigKey := fmt.Sprintf("%s/wechat/%d", key, i)

		if len(config.APIURL) > 0 {
			_, err := url.Parse(config.APIURL)
			if err != nil {
				return errors.New("API URL not valid")
			}
		}

		if config.APISecret != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.APISecret); err != nil {
				return err
			}
		}

		if err := configureHTTPConfigInStore(ctx, config.HTTPConfig, namespace, wechatConfigKey, store); err != nil {
			return err
		}
	}

	return nil
}

func checkEmailConfigs(ctx context.Context, configs []v1alpha1.EmailConfig, namespace string, _ string, store *assets.Store) error {
	for _, config := range configs {
		if config.To == "" {
			return errors.New("missing to address in email config")
		}

		if config.Smarthost != "" {
			_, _, err := net.SplitHostPort(config.Smarthost)
			if err != nil {
				return errors.New("invalid email field SMARTHOST")
			}
		}
		if config.AuthPassword != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.AuthPassword); err != nil {
				return err
			}
		}
		if config.AuthSecret != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.AuthSecret); err != nil {
				return err
			}
		}

		if config.Headers != nil {
			// Header names are case-insensitive, check for collisions.
			normalizedHeaders := map[string]struct{}{}
			for _, v := range config.Headers {
				normalized := strings.Title(v.Key)
				if _, ok := normalizedHeaders[normalized]; ok {
					return fmt.Errorf("duplicate header %q in email config", normalized)
				}
				normalizedHeaders[normalized] = struct{}{}
			}
		}

		if err := store.AddSafeTLSConfig(ctx, namespace, config.TLSConfig); err != nil {
			return err
		}
	}

	return nil
}

func checkVictorOpsConfigs(ctx context.Context, configs []v1alpha1.VictorOpsConfig, namespace string, key string, store *assets.Store) error {
	for i, config := range configs {
		if config.APIKey != nil {
			if _, err := store.GetSecretKey(ctx, namespace, *config.APIKey); err != nil {
				return err
			}
		}

		// from https://github.com/prometheus/alertmanager/blob/a7f9fdadbecbb7e692d2cd8d3334e3d6de1602e1/config/notifiers.go#L497
		reservedFields := map[string]struct{}{
			"routing_key":         {},
			"message_type":        {},
			"state_message":       {},
			"entity_display_name": {},
			"monitoring_tool":     {},
			"entity_id":           {},
			"entity_state":        {},
		}

		if len(config.CustomFields) > 0 {
			for _, v := range config.CustomFields {
				if _, ok := reservedFields[v.Key]; ok {
					return fmt.Errorf("usage of reserved word %q is not allowed in custom fields", v.Key)
				}
			}
		}

		if config.RoutingKey == "" {
			return errors.New("missing Routing key in VictorOps config")
		}

		victoropsConfigKey := fmt.Sprintf("%s/victorops/%d", key, i)
		if err := configureHTTPConfigInStore(ctx, config.HTTPConfig, namespace, victoropsConfigKey, store); err != nil {
			return err
		}
	}

	return nil
}

func checkPushoverConfigs(ctx context.Context, configs []v1alpha1.PushoverConfig, namespace string, key string, store *assets.Store) error {
	checkSecret := func(secret *v1.SecretKeySelector, name string) error {
		if secret == nil {
			return errors.Errorf("mandatory field %s is empty", name)
		}
		s, err := store.GetSecretKey(ctx, namespace, *secret)
		if err != nil {
			return err
		}
		if s == "" {
			return errors.New("mandatory field userKey is empty")
		}
		return nil
	}

	for i, config := range configs {
		if err := checkSecret(config.UserKey, "userKey"); err != nil {
			return err
		}
		if err := checkSecret(config.Token, "token"); err != nil {
			return err
		}

		if config.Retry != "" {
			_, err := time.ParseDuration(config.Retry)
			if err != nil {
				return errors.New("invalid retry duration")
			}
		}
		if config.Expire != "" {
			_, err := time.ParseDuration(config.Expire)
			if err != nil {
				return errors.New("invalid expire duration")
			}
		}

		pushoverConfigKey := fmt.Sprintf("%s/pushover/%d", key, i)
		if err := configureHTTPConfigInStore(ctx, config.HTTPConfig, namespace, pushoverConfigKey, store); err != nil {
			return err
		}
	}

	return nil
}

// checkAlertmanagerRoutes verifies that the given route and all its children are semantically valid.
func checkAlertmanagerRoutes(r *v1alpha1.Route, receivers map[string]struct{}, topLevelRoute bool) error {
	if r == nil {
		return nil
	}

	if _, found := receivers[r.Receiver]; !found && (r.Receiver != "" || topLevelRoute) {
		return errors.Errorf("receiver %q not found", r.Receiver)
	}

	children, err := r.ChildRoutes()
	if err != nil {
		return err
	}

	for i := range children {
		if err := checkAlertmanagerRoutes(&children[i], receivers, false); err != nil {
			return errors.Wrapf(err, "route[%d]", i)
		}
	}

	return nil
}

// configureHTTPConfigInStore configure the asset store for HTTPConfigs.
func configureHTTPConfigInStore(ctx context.Context, httpConfig *v1alpha1.HTTPConfig, namespace string, key string, store *assets.Store) error {
	if httpConfig == nil {
		return nil
	}

	var err error
	if httpConfig.BearerTokenSecret != nil {
		if err = store.AddBearerToken(ctx, namespace, *httpConfig.BearerTokenSecret, key); err != nil {
			return err
		}
	}

	if err = store.AddBasicAuth(ctx, namespace, httpConfig.BasicAuth, key); err != nil {
		return err
	}

	if err = store.AddSafeTLSConfig(ctx, namespace, httpConfig.TLSConfig); err != nil {
		return err
	}
	return nil
}
