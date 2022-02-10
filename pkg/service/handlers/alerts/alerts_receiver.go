package alerthandler

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/gin-gonic/gin"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/kubeclient"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/prometheus"
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

type ReceiverConfigs []ReceiverConfig

func (a ReceiverConfigs) Len() int      { return len(a) }
func (a ReceiverConfigs) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ReceiverConfigs) Less(i, j int) bool {
	return strings.ToLower(a[i].Name) < strings.ToLower(a[j].Name)
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
	ctx := c.Request.Context()

	ret := ReceiverConfigs{}
	if namespace == allNamespace {
		configs, err := kubeclient.GetClient().GetAlertmanagerConfigList(cluster, v1.NamespaceAll, prometheus.AlertmanagerConfigSelector)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		secrets, err := kubeclient.GetClient().GetSecretList(cluster, corev1.NamespaceAll, emailSecretLabel)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}

		for _, config := range *configs {
			secret := filterSecretByNamespace(secrets, config.Namespace)
			for _, rec := range config.Spec.Receivers {
				if rec.Name != nullReceiverName {
					if search == "" || (search != "" && strings.Contains(rec.Name, search)) {
						ret = append(ret, toGemsReceiver(rec, config.Namespace, secret))
					}
				}
			}
		}
	} else {
		config, err := getOrCreateAlertmanagerConfig(ctx, cluster, namespace)
		if err != nil {
			handlers.NotOK(c, err)
			return
		}
		secret, err := kubeclient.GetClient().GetSecret(cluster, namespace, emailSecretName, nil)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				handlers.NotOK(c, err)
				return
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

	sort.Sort(ret)
	handlers.OK(c, ret)
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
// @Success 200 {object} handlers.ResponseStruct{Data=[]v1alpha1.Receiver} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver [post]
// @Security JWT
func (h *AlertmanagerConfigHandler) CreateReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	ctx := c.Request.Context()
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "创建", "告警接收器", "")
	aconfig, err := getOrCreateAlertmanagerConfig(ctx, cluster, namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	req := ReceiverConfig{}
	_ = c.BindJSON(&req)
	h.SetAuditData(c, "创建", "告警接收器", req.Name)

	if err := req.Precheck(); err != nil {
		handlers.NotOK(c, err)
		return
	}

	if err := req.createOrUpdateSecret(cluster, namespace); err != nil {
		handlers.NotOK(c, err)
		return
	}

	receiver := toAlertmanagerReceiver(req)
	if err := h.modify(aconfig, &receiver, cluster, prometheus.Add); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, aconfig.Spec.Receivers)
}

// @Tags Alert
// @Summary 在namespace下删除receiver
// @Description 在namespace下创建receiver
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param namespace path string true "namespace"
// @Param name path string true "name"
// @Success 200 {object} handlers.ResponseStruct{Data=[]v1alpha1.Receiver} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name} [delete]
// @Security JWT
func (h *AlertmanagerConfigHandler) DeleteReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")

	if name == defaultReceiverName {
		handlers.NotOK(c, fmt.Errorf("不能删除默认接收器"))
		return
	}

	ctx := c.Request.Context()
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "删除", "告警接收器", name)
	if len(name) == 0 {
		handlers.NotOK(c, fmt.Errorf("receiver name must not be empty"))
		return
	}
	aconfig, err := getOrCreateAlertmanagerConfig(ctx, cluster, namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	receiver := v1alpha1.Receiver{Name: name}
	if isReceiverUsed(aconfig.Spec.Route, receiver) {
		handlers.NotOK(c, fmt.Errorf("%s is being used, can't delete", name))
		return
	}
	if err := h.modify(aconfig, &receiver, cluster, prometheus.Delete); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, aconfig.Spec.Receivers)
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
// @Success 200 {object} handlers.ResponseStruct{Data=[]v1alpha1.Receiver} "告警通知接收人"
// @Router /v1/alerts/cluster/{cluster}/namespaces/{namespace}/receiver/{name} [put]
// @Security JWT
func (h *AlertmanagerConfigHandler) ModifyReceiver(c *gin.Context) {
	cluster := c.Param("cluster")
	namespace := c.Param("namespace")
	name := c.Param("name")
	ctx := c.Request.Context()
	h.SetExtraAuditDataByClusterNamespace(c, cluster, namespace)
	h.SetAuditData(c, "修改", "告警接收器", name)
	aconfig, err := getOrCreateAlertmanagerConfig(ctx, cluster, namespace)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	req := ReceiverConfig{}
	_ = c.BindJSON(&req)
	if err := req.Precheck(); err != nil {
		handlers.NotOK(c, err)
		return
	}
	if err := req.createOrUpdateSecret(cluster, namespace); err != nil {
		handlers.NotOK(c, err)
		return
	}

	receiver := toAlertmanagerReceiver(req)
	if err := h.modify(aconfig, &receiver, cluster, prometheus.Update); err != nil {
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, aconfig.Spec.Receivers)
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

func (h *AlertmanagerConfigHandler) modify(aconfig *v1alpha1.AlertmanagerConfig, receiver *v1alpha1.Receiver, cluster string, act prometheus.Action) error {
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
		_, err := kubeclient.GetClient().UpdateAlertmanagerConfig(cluster, aconfig)
		return err
	case prometheus.Delete:
		index := findReceiverIndex(aconfig.Spec.Receivers, receiver.Name)
		if index != -1 {
			toDelete := aconfig.Spec.Receivers[index]
			aconfig.Spec.Receivers = append(aconfig.Spec.Receivers[:index], aconfig.Spec.Receivers[index+1:]...)
			if _, err := kubeclient.GetClient().UpdateAlertmanagerConfig(cluster, aconfig); err != nil {
				return err
			}

			return deleteSecret(cluster, aconfig.Namespace, toDelete)
		}
		return nil
	case prometheus.Update:
		index := findReceiverIndex(aconfig.Spec.Receivers, receiver.Name)
		if index == -1 {
			return fmt.Errorf("receiver %s not exist", receiver.Name)
		}
		aconfig.Spec.Receivers[index] = *receiver
		aconfig.Spec.Route.Receiver = nullReceiverName
		_, err := kubeclient.GetClient().UpdateAlertmanagerConfig(cluster, aconfig)
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

func (rec *ReceiverConfig) createOrUpdateSecret(cluster, namespace string) error {
	var m sync.Mutex
	m.Lock()
	defer m.Unlock()

	sec, err := kubeclient.GetClient().GetSecret(cluster, namespace, emailSecretName, nil)
	if err != nil {
		if kerrors.IsNotFound(err) {
			if _, err := kubeclient.GetClient().CreateSecret(cluster, namespace, emailSecretName, &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Name:      emailSecretName,
					Namespace: namespace,
					Labels:    emailSecretLabel,
				},
				Type: corev1.SecretTypeOpaque,
				Data: rec.secretData(),
			}); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if sec.Data == nil {
		sec.Data = make(map[string][]byte)
	}
	for k, v := range rec.secretData() {
		sec.Data[k] = v
	}

	if _, err := kubeclient.GetClient().PatchSecret(cluster, namespace, emailSecretName, sec); err != nil {
		return err
	}
	return nil
}

func deleteSecret(cluster, namespace string, rec v1alpha1.Receiver) error {
	sec, err := kubeclient.GetClient().GetSecret(cluster, namespace, emailSecretName, nil)
	if err != nil {
		return err
	}

	for _, v := range rec.EmailConfigs {
		delete(sec.Data, emailSecretKey(rec.Name, v.From))
	}

	_, err = kubeclient.GetClient().PatchSecret(cluster, namespace, emailSecretName, sec)
	return err
}

func emailSecretKey(receverName, from string) string {
	return receverName + "-" + strings.ReplaceAll(from, "@", "")
}

func (rec *ReceiverConfig) secretData() map[string][]byte {
	ret := map[string][]byte{}
	for _, v := range rec.EmailConfigs {
		ret[emailSecretKey(rec.Name, v.From)] = []byte(v.AuthPassword) // 不需要encode
	}
	return ret
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
