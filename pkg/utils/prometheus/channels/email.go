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
	"strings"
	"time"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	v1 "k8s.io/api/core/v1"
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
			},
		},
	}
}

func (e *Email) Check() error {
	return nil
}

func (e *Email) Test() error {
	auth := sasl.NewPlainClient("", e.From, e.AuthPassword)
	receivers := strings.Split(e.To, ",")
	msg := strings.NewReader("To: " + e.To + "\r\n" +
		"Subject: Kubegems test email" + "\r\n" +
		"\r\n" +
		" at " + time.Now().Format("2006-01-02 15:04:05"))
	return smtp.SendMail(e.SMTPServer, auth, e.From, receivers, msg)
}
