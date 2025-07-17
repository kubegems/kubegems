// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package channels

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type Email struct {
	BaseChannel  `json:",inline"`
	SMTPServer   string `json:"smtpServer" binding:"required"`
	RequireTLS   bool   `json:"requireTLS"`
	From         string `json:"from" binding:"required"`
	To           string `json:"to" binding:"required"`
	AuthPassword string `json:"authPassword" binding:"required"`
}

var (
	EmailSecretName       = "gemscloud-email-password"
	EmailSecretLabelKey   = "gemcloud"
	EmailSecretLabelValue = "email-secret"
)

func EmailSecretKey(receverName, from string) string {
	return receverName + "-" + strings.ReplaceAll(from, "@", "")
}

func (e *Email) ToReceiver(name string) v1alpha1.Receiver {
	return v1alpha1.Receiver{
		Name: name,
		EmailConfigs: []v1alpha1.EmailConfig{
			{
				Smarthost:    e.SMTPServer,
				RequireTLS:   &e.RequireTLS,
				From:         e.From,
				AuthUsername: e.From,
				AuthIdentity: e.From,
				To:           e.To,
				AuthPassword: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: EmailSecretName,
					},
					Key: EmailSecretKey(name, e.From),
				},
				HTML: `{{ template "email.common.html" . }}`,
				Headers: []v1alpha1.KeyValue{
					{
						Key:   "subject",
						Value: `Kubegems alert [{{ .CommonLabels.gems_alertname }}:{{ .Alerts.Firing | len }}] in [cluster:{{ .CommonLabels.cluster }}] [namespace:{{ .CommonLabels.gems_namespace }}]`,
					},
				},
				SendResolved: utils.BoolPointer(e.SendResolved),
			},
		},
	}
}

func (e *Email) Check() error {
	return nil
}

func (e *Email) String() string {
	return e.SMTPServer + e.From + e.To
}

// SendEmail 通用的邮件发送方法
func (e *Email) Test(alert prometheus.WebhookAlert) error {
	// 验证必填字段
	if e.SMTPServer == "" || e.From == "" || e.To == "" || e.AuthPassword == "" {
		return fmt.Errorf("missing required email parameters")
	}
	body, _ := json.MarshalIndent(alert, "", "    ")
	// 设置默认端口
	host, port := splitServerAddress(e.SMTPServer)
	// 构建邮件内容
	message := fmt.Sprintf("From: %s\r\n", e.From) +
		fmt.Sprintf("To: %s\r\n", e.To) +
		"Subject: Kubegems test email\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		string(body)

	// 认证信息
	auth := smtp.PlainAuth("", e.From, e.AuthPassword, host)
	// 收件人列表
	to := strings.Split(e.To, ",")
	for i := range to {
		to[i] = strings.TrimSpace(to[i])
	}
	// 发送邮件
	if e.RequireTLS {
		// 使用TLS连接
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         host,
		}

		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%s", host, port), tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP server with TLS: %v", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %v", err)
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}

		if err = client.Mail(e.From); err != nil {
			return fmt.Errorf("failed to set sender: %v", err)
		}

		for _, addr := range to {
			if err = client.Rcpt(addr); err != nil {
				return fmt.Errorf("failed to set recipient %s: %v", addr, err)
			}
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %v", err)
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to write message: %v", err)
		}

		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close data writer: %v", err)
		}

		return client.Quit()
	} else {
		// 普通连接
		return smtp.SendMail(
			fmt.Sprintf("%s:%s", host, port),
			auth,
			e.From,
			to,
			[]byte(message),
		)
	}
}

// splitServerAddress 分离服务器地址和端口
func splitServerAddress(server string) (host, port string) {
	parts := strings.Split(server, ":")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return server, "25" // 默认SMTP端口
}
