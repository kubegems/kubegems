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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
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
func (e *Email) message(alert prometheus.WebhookAlert) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", e.From)
	fmt.Fprintf(&b, "To: %s\r\n", e.To)
	fmt.Fprintf(&b, "Subject: Kubegems test email\r\n")
	fmt.Fprintf(&b, "\r\n")
	data, _ := json.MarshalIndent(alert, "", "    ")
	b.Write(data)
	return b.Bytes()
}

func (e *Email) Test(alert prometheus.WebhookAlert) error {
	if e.SMTPServer == "" {
		return fmt.Errorf("smtp address is required")
	}
	var smtpServer = e.SMTPServer
	if !strings.Contains(e.SMTPServer, "//") {
		smtpServer = "smtp://" + e.SMTPServer
		if e.RequireTLS {
			smtpServer = "smtps://" + e.SMTPServer
		}
	}
	u, err := url.Parse(smtpServer)
	if err != nil {
		return fmt.Errorf("invalid smtp address: %v", err)
	}
	cli, err := smtp.Dial(u.Host)
	if err != nil {
		return err
	}
	if u.Scheme == "smtps" {
		if ok, _ := cli.Extension("STARTTLS"); !ok {
			return fmt.Errorf("server does not support tls, but tls is required")
		}
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		if err := cli.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		return err
	}
	if e.From != "" && e.AuthPassword != "" {
		auth := smtp.PlainAuth("", e.From, e.AuthPassword, host)
		if ok, _ := cli.Extension("AUTH"); !ok {
			return fmt.Errorf("server does not support auth, but username and password are provided")
		}
		if err := cli.Auth(auth); err != nil {
			return err
		}
	}

	if err := cli.Mail(e.From); err != nil {
		return err
	}
	for _, v := range strings.Split(e.To, ",") {
		if err := cli.Rcpt(v); err != nil {
			return err
		}
	}
	w, err := cli.Data()
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = w.Write(e.message(alert))
	return err
}
