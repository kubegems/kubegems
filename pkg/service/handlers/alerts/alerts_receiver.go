package alerthandler

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/gin-gonic/gin"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	emailSecretName = "gemscloud-email-password"
)

var emailSecretLabel map[string]string = map[string]string{
	"gemcloud": "email-secret",
}

type EmailConfig struct {
	SMTPServer   string `json:"smtpServer"`
	RequireTLS   bool   `json:"requireTLS"`
	From         string `json:"from"`
	To           string `json:"to"`
	AuthPassword string `json:"authPassword"`
}

type WebhookConfig struct {
	URL string `json:"url"`
}

type ReceiverConfig struct {
	Name           string          `json:"name"`
	Namespace      string          `json:"namespace"`
	EmailConfigs   []EmailConfig   `json:"emailConfigs"`
	WebhookConfigs []WebhookConfig `json:"webhookConfigs"`
}

func (rec *ReceiverConfig) Precheck() error {
	for _, v := range rec.WebhookConfigs {
		if _, err := url.ParseRequestURI(v.URL); err != nil {
			return fmt.Errorf("URL: %s 不合法, %w", v.URL, err)
		}
	}
	if rec.Name == defaultReceiverName {
		return fmt.Errorf("不能修改默认接收器")
	}
	return nil
}

// @Tags Alert
// @Summary 在namespace下获取receiver列表
// @Description 在namespace下获取receiver列表
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param search query string true "search"
// @Success 200 {object} handlers.ResponseStruct{Data=[]ReceiverConfig} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver [get]
// @Security JWT
func (h *AlertmanagerConfigHandler) ListReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	search := c.Query("search")

	ret := []ReceiverConfig{}
	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		if namespace == allNamespace {
			configlist := &v1alpha1.AlertmanagerConfigList{}
			emailsecretlist := &corev1.SecretList{}

			if err := cli.List(ctx, configlist, client.MatchingLabels(prometheus.AlertmanagerConfigSelector)); err != nil {
				return nil, err
			}
			if err := cli.List(ctx, emailsecretlist, client.MatchingLabels(emailSecretLabel)); err != nil {
				return nil, err
			}

			for _, config := range configlist.Items {
				secret := filterSecretByNamespace(&emailsecretlist.Items, config.Namespace)
				for _, rec := range config.Spec.Receivers {
					if rec.Name != nullReceiverName {
						if search == "" || (search != "" && strings.Contains(rec.Name, search)) {
							ret = append(ret, toGemsReceiver(rec, config.Namespace, secret))
						}
					}
				}
			}
		} else {
			config, err := cli.Extend().GetOrCreateAlertmanagerConfig(ctx, namespace)
			if err != nil {
				return nil, err
			}

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      emailSecretName,
					Namespace: namespace,
				},
			}
			if err := cli.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
				if !kerrors.IsNotFound(err) {
					return err, nil
				}
			}

			for _, rec := range config.Spec.Receivers {
				if rec.Name != nullReceiverName {
					if search == "" || (search != "" && strings.Contains(rec.Name, search)) {
						ret = append(ret, toGemsReceiver(rec, config.Namespace, secret))
					}
				}
			}
		}

		sort.Slice(ret, func(i, j int) bool {
			return strings.ToLower(ret[i].Name) < strings.ToLower(ret[j].Name)
		})
		return ret, nil
	})(c)

}

func filterSecretByNamespace(secrets *[]corev1.Secret, ns string) *corev1.Secret {
	for _, v := range *secrets {
		if v.Namespace == ns && v.Name == emailSecretName {
			return &v
		}
	}
	return nil
}

// @Tags Alert
// @Summary 在namespace下创建receiver
// @Description 在namespace下创建receiver
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param form body ReceiverConfig true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver [post]
// @Security JWT
func (h *AlertmanagerConfigHandler) CreateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		aconfig, err := cli.Extend().GetOrCreateAlertmanagerConfig(ctx, namespace)
		if err != nil {
			return nil, err
		}
		req := ReceiverConfig{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}
		h.SetAuditData(c, "创建", "告警接收器", req.Name)

		if err := req.Precheck(); err != nil {
			return nil, err
		}

		if err := createOrUpdateSecret(c.Request.Context(), namespace, &req, cli); err != nil {
			return nil, err
		}

		receiver := toAlertmanagerReceiver(req)
		if err := h.modify(ctx, aconfig, &receiver, prometheus.Add, cli); err != nil {
			return nil, err
		}
		return "ok", err
	})(c)
}

// @Tags Alert
// @Summary 在namespace下删除receiver
// @Description 在namespace下创建receiver
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name} [delete]
// @Security JWT
func (h *AlertmanagerConfigHandler) DeleteReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "删除", "告警接收器", name)

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		if name == defaultReceiverName {
			return nil, fmt.Errorf("不能删除默认接收器")
		}
		if len(name) == 0 {
			return nil, fmt.Errorf("receiver name must not be empty")
		}
		aconfig, err := cli.Extend().GetOrCreateAlertmanagerConfig(ctx, namespace)
		if err != nil {
			return nil, err
		}
		receiver := v1alpha1.Receiver{Name: name}
		if isReceiverUsed(aconfig.Spec.Route, receiver) {
			return nil, fmt.Errorf("%s is being used, can't delete", name)
		}
		if err := h.modify(ctx, aconfig, &receiver, prometheus.Delete, cli); err != nil {
			return nil, err
		}
		return "ok", err
	})(c)
}

// @Tags Alert
// @Summary 在namespace下修改receiver
// @Description 在namespace下修改receiver
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Param form body ReceiverConfig true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name} [put]
// @Security JWT
func (h *AlertmanagerConfigHandler) ModifyReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)

	h.ClusterFunc(cluster, func(ctx context.Context, cli agents.Client) (interface{}, error) {
		aconfig, err := cli.Extend().GetOrCreateAlertmanagerConfig(ctx, namespace)
		if err != nil {
			return nil, err
		}
		req := ReceiverConfig{}
		if err := c.BindJSON(&req); err != nil {
			return nil, err
		}
		h.SetAuditData(c, "修改", "告警接收器", req.Name)

		if err := req.Precheck(); err != nil {
			return nil, err
		}

		if err := createOrUpdateSecret(c.Request.Context(), namespace, &req, cli); err != nil {
			return nil, err
		}

		receiver := toAlertmanagerReceiver(req)
		if err := h.modify(ctx, aconfig, &receiver, prometheus.Update, cli); err != nil {
			return nil, err
		}
		return "ok", err
	})(c)
}

// @Tags Alert
// @Summary 发送测试邮件
// @Description 发送测试邮件
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Param form body EmailConfig true "body"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name}/actions/test [post]
// @Security JWT
func (h *AlertmanagerConfigHandler) TestEmail(c *gin.Context) {
	req := EmailConfig{}
	c.BindJSON(&req)
	h.SetExtraAuditDataByClusterNamespace(c, c.Param("cluster"), c.Param("namespace"))
	h.SetAuditData(c, "测试", "告警接收器", "")

	if err := req.testEmail(c.Param("cluster"), c.Param("namespace")); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "邮件发送成功！")
}

func (h *AlertmanagerConfigHandler) AlertmanagerConfigName(environment *models.Environment) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(environment.Project.ProjectName), strings.ToLower(environment.EnvironmentName))
}

func findReceiverIndex(rules []v1alpha1.Receiver, name string) int {
	index := -1
	for idx := range rules {
		if rules[idx].Name == name {
			index = idx
		}
	}
	return index
}

func (h *AlertmanagerConfigHandler) modify(ctx context.Context, aconfig *v1alpha1.AlertmanagerConfig, receiver *v1alpha1.Receiver, act prometheus.Action, cli agents.Client) error {
	if receiver == nil {
		return nil
	}
	// 更改邮件模板
	for i := range receiver.EmailConfigs {
		receiver.EmailConfigs[i].HTML = `{{ template "email.common.html" . }}`
		receiver.EmailConfigs[i].Headers = []v1alpha1.KeyValue{
			{
				Key:   "subject",
				Value: `[{{ .CommonLabels.gems_alertname }}:{{ .Alerts.Firing | len }}] created by kubegems in [cluster:{{ .CommonLabels.cluster }}] [namespace:{{ .CommonLabels.gems_namespace }}]`,
			},
		}
	}

	h.m.Lock()
	defer h.m.Unlock()

	switch act {
	case prometheus.Add:
		index := findReceiverIndex(aconfig.Spec.Receivers, receiver.Name)
		if index != -1 {
			return fmt.Errorf("receiver %s existed", receiver.Name)
		}
		aconfig.Spec.Receivers = append(aconfig.Spec.Receivers, *receiver)
		aconfig.Spec.Route.Receiver = nullReceiverName
		_, err := updateAlertmanagerConfig(ctx, aconfig, cli)
		return err
	case prometheus.Delete:
		index := findReceiverIndex(aconfig.Spec.Receivers, receiver.Name)
		if index != -1 {
			toDelete := aconfig.Spec.Receivers[index]
			aconfig.Spec.Receivers = append(aconfig.Spec.Receivers[:index], aconfig.Spec.Receivers[index+1:]...)
			if _, err := updateAlertmanagerConfig(ctx, aconfig, cli); err != nil {
				return err
			}

			return deleteSecret(ctx, aconfig.Namespace, toDelete, cli)
		}
		return nil
	case prometheus.Update:
		index := findReceiverIndex(aconfig.Spec.Receivers, receiver.Name)
		if index == -1 {
			return fmt.Errorf("receiver %s not exist", receiver.Name)
		}
		aconfig.Spec.Receivers[index] = *receiver
		aconfig.Spec.Route.Receiver = nullReceiverName
		_, err := updateAlertmanagerConfig(ctx, aconfig, cli)
		return err
	}
	return nil
}

func isReceiverUsed(route *v1alpha1.Route, receiver v1alpha1.Receiver) bool {
	if route.Receiver == receiver.Name {
		return true
	}
	children, e := route.ChildRoutes()
	if e != nil {
		return false
	}
	for _, r := range children {
		if isReceiverUsed(&r, receiver) {
			return true
		}
	}
	return false
}

func createOrUpdateSecret(ctx context.Context, namespace string, rec *ReceiverConfig, cli agents.Client) error {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      emailSecretName,
			Namespace: namespace,
			Labels:    emailSecretLabel,
		},
		Type: corev1.SecretTypeOpaque,
	}
	_, err := controllerutil.CreateOrUpdate(ctx, cli, sec, func() error {
		if sec.Data == nil {
			sec.Data = make(map[string][]byte)
		}
		for _, v := range rec.EmailConfigs {
			sec.Data[emailSecretKey(rec.Name, v.From)] = []byte(v.AuthPassword) // 不需要encode
		}
		return nil
	})
	return err
}

func deleteSecret(ctx context.Context, namespace string, rec v1alpha1.Receiver, cli agents.Client) error {
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      emailSecretName,
			Namespace: namespace,
		},
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(sec), sec); err != nil {
		return err
	}
	for _, v := range rec.EmailConfigs {
		delete(sec.Data, emailSecretKey(rec.Name, v.From))
	}
	return cli.Update(ctx, sec)
}

func emailSecretKey(receverName, from string) string {
	return receverName + "-" + strings.ReplaceAll(from, "@", "")
}

func toGemsReceiver(rec v1alpha1.Receiver, namespace string, sec *corev1.Secret) ReceiverConfig {
	ret := ReceiverConfig{
		Name:      rec.Name,
		Namespace: namespace,
	}

	if sec != nil {
		for _, v := range rec.EmailConfigs {
			ret.EmailConfigs = append(ret.EmailConfigs, EmailConfig{
				SMTPServer:   v.Smarthost,
				RequireTLS:   *v.RequireTLS,
				From:         v.From,
				AuthPassword: string(sec.Data[emailSecretKey(rec.Name, v.From)]),
				To:           v.To,
			})
		}
	}

	for _, v := range rec.WebhookConfigs {
		ret.WebhookConfigs = append(ret.WebhookConfigs, WebhookConfig{
			URL: *v.URL,
		})
	}
	return ret
}

func toAlertmanagerReceiver(rec ReceiverConfig) v1alpha1.Receiver {
	ret := v1alpha1.Receiver{
		Name: rec.Name,
	}
	// 涉及指针赋值，不能用v range,因为每次v及其值的地址是同一个，必须用index遍历
	for i := range rec.EmailConfigs {
		ret.EmailConfigs = append(ret.EmailConfigs, v1alpha1.EmailConfig{
			Smarthost:    rec.EmailConfigs[i].SMTPServer,
			RequireTLS:   &rec.EmailConfigs[i].RequireTLS,
			From:         rec.EmailConfigs[i].From,
			AuthUsername: rec.EmailConfigs[i].From,
			AuthIdentity: rec.EmailConfigs[i].From,
			To:           rec.EmailConfigs[i].To,
			AuthPassword: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: emailSecretName,
				},
				Key: emailSecretKey(rec.Name, rec.EmailConfigs[i].From),
			},
		})
	}
	for i := range rec.WebhookConfigs {
		ret.WebhookConfigs = append(ret.WebhookConfigs, v1alpha1.WebhookConfig{
			URL: &rec.WebhookConfigs[i].URL,
		})
	}
	return ret
}

func (e EmailConfig) testEmail(cluster, namespace string) error {
	auth := sasl.NewPlainClient("", e.From, e.AuthPassword)
	receivers := strings.Split(e.To, ",")
	msg := strings.NewReader("To: " + e.To + "\r\n" +
		"Subject: gemscloud test email" + "\r\n" +
		"\r\n" +
		"from cluster " + cluster + " namespace " + namespace + " at " + time.Now().Format("2006-01-02 15:04:05"))
	return smtp.SendMail(e.SMTPServer, auth, e.From, receivers, msg)
}
