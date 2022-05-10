package prometheus

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

var (
	DefaultReceiverName = "gemcloud-default-webhook"
	DefaultReceiverURL  = "https://gems-agent.gemcloud-system:8041/alert"
	DefaultReceiver     = v1alpha1.Receiver{
		Name: DefaultReceiverName,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL: &DefaultReceiverURL,
				HTTPConfig: &v1alpha1.HTTPConfig{
					TLSConfig: &v1.SafeTLSConfig{
						InsecureSkipVerify: true,
					},
				},
			},
		},
	}

	NullReceiverName = "null"
	NullReceiver     = v1alpha1.Receiver{Name: NullReceiverName}

	EmailSecretName                    = "gemscloud-email-password"
	EmailSecretLabel map[string]string = map[string]string{
		"gemcloud": "email-secret",
	}
)

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
	if rec.Name == DefaultReceiverName {
		return fmt.Errorf("不能修改默认接收器")
	}
	return nil
}

func TestEmail(e EmailConfig, cluster, namespace string) error {
	auth := sasl.NewPlainClient("", e.From, e.AuthPassword)
	receivers := strings.Split(e.To, ",")
	msg := strings.NewReader("To: " + e.To + "\r\n" +
		"Subject: Kubegems test email" + "\r\n" +
		"\r\n" +
		"from cluster " + cluster + " namespace " + namespace + " at " + time.Now().Format("2006-01-02 15:04:05"))
	return smtp.SendMail(e.SMTPServer, auth, e.From, receivers, msg)
}

func ModifyReceiver(ctx context.Context, aconfig *v1alpha1.AlertmanagerConfig, receiver *v1alpha1.Receiver, act Action) error {
	if receiver == nil {
		return nil
	}
	if receiver.Name == "" {
		return fmt.Errorf("receiver name must not be empty")
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
	index := findReceiverIndex(aconfig.Spec.Receivers, receiver.Name)
	switch act {
	case Add:
		if index != -1 {
			return fmt.Errorf("receiver %s existed", receiver.Name)
		}
		aconfig.Spec.Receivers = append(aconfig.Spec.Receivers, *receiver)
		aconfig.Spec.Route.Receiver = NullReceiverName
	case Delete:
		if receiver.Name == DefaultReceiverName {
			return fmt.Errorf("不能删除默认接收器")
		}
		if isReceiverInUse(aconfig.Spec.Route, *receiver) {
			return fmt.Errorf("%s is being used, can't delete", receiver.Name)
		}
		if index == -1 {
			return fmt.Errorf("receiver %s not exist", receiver.Name)
		}
		// 删除前记录要删除的recever
		receiver = &aconfig.Spec.Receivers[index]
		aconfig.Spec.Receivers = append(aconfig.Spec.Receivers[:index], aconfig.Spec.Receivers[index+1:]...)
	case Update:
		if index == -1 {
			return fmt.Errorf("receiver %s not exist", receiver.Name)
		}
		aconfig.Spec.Receivers[index] = *receiver
		aconfig.Spec.Route.Receiver = NullReceiverName
	}
	return nil
}

func EmailSecretKey(receverName, from string) string {
	return receverName + "-" + strings.ReplaceAll(from, "@", "")
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

func ToGemsReceiver(rec v1alpha1.Receiver, namespace string, sec *corev1.Secret) ReceiverConfig {
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
				AuthPassword: string(sec.Data[EmailSecretKey(rec.Name, v.From)]),
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

func ToAlertmanagerReceiver(rec ReceiverConfig) v1alpha1.Receiver {
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
					Name: EmailSecretName,
				},
				Key: EmailSecretKey(rec.Name, rec.EmailConfigs[i].From),
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

func isReceiverInUse(route *v1alpha1.Route, receiver v1alpha1.Receiver) bool {
	if route.Receiver == receiver.Name {
		return true
	}
	children, e := route.ChildRoutes()
	if e != nil {
		return false
	}
	for _, r := range children {
		if isReceiverInUse(&r, receiver) {
			return true
		}
	}
	return false
}
