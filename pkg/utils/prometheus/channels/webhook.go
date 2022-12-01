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
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type Webhook struct {
	BaseChannel        `json:",inline"`
	URL                string `json:"url" binding:"required"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
}

func (w *Webhook) ToReceiver(name string) v1alpha1.Receiver {
	cfg := v1alpha1.WebhookConfig{
		URL:          &w.URL,
		SendResolved: utils.BoolPointer(w.SendResolved),
	}
	if w.InsecureSkipVerify {
		cfg.HTTPConfig = &v1alpha1.HTTPConfig{
			TLSConfig: &monv1.SafeTLSConfig{
				InsecureSkipVerify: true,
			},
		}
	}
	return v1alpha1.Receiver{
		Name:           name,
		WebhookConfigs: []v1alpha1.WebhookConfig{cfg},
	}
}

func (w *Webhook) Check() error {
	if _, err := url.ParseRequestURI(w.URL); err != nil {
		return errors.Wrap(err, "url 不合法")
	}
	return nil
}

func (w *Webhook) Test(alert prometheus.WebhookAlert) error {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(alert); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, w.URL, buf)
	if err != nil {
		return err
	}
	testCli := &http.Client{}
	if w.InsecureSkipVerify {
		testCli.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	resp, err := testCli.Do(req)
	if err != nil {
		return err
	}
	bts, _ := io.ReadAll(resp.Body)
	log.Info("test webhook success", "url", w.URL, "resp", string(bts))
	return nil
}

func (w *Webhook) String() string {
	return w.URL
}
