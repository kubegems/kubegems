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
	"encoding/json"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type Email struct {
	ChannelType  `json:"channelType"`
	SMTPServer   string `json:"smtpServer"`
	RequireTLS   bool   `json:"requireTLS"`
	From         string `json:"from"`
	To           string `json:"to"`
	AuthPassword string `json:"authPassword"`
}

var (
	EmailSecretName                    = "gemscloud-email-password"
	EmailSecretLabel map[string]string = map[string]string{
		"gemcloud": "email-secret",
	}
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
			},
		},
	}
}

func (e *Email) Check() error {
	return nil
}

func (e *Email) Test(alert prometheus.WebhookAlert) error {
	auth := sasl.NewPlainClient("", e.From, e.AuthPassword)
	receivers := strings.Split(e.To, ",")
	buf := bytes.NewBufferString("To: " + e.To + "\r\n" +
		"Subject: Kubegems test email" + "\r\n" +
		"\r\n")

	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(alert); err != nil {
		return err
	}
	return smtp.SendMail(e.SMTPServer, auth, e.From, receivers, buf)
}
