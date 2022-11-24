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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

var (
	alertProxyReceiverHost = fmt.Sprintf("alertproxy.%s:9094", gems.NamespaceMonitor)
	alertProxyFeishu       = "feishu"
)

type Feishu struct {
	BaseChannel `json:",inline"`
	URL         string `json:"url"`        // feishu robot webhook url
	At          string `json:"at"`         // 要@的用户id，所有人则是 all
	SignSecret  string `json:"signSecret"` // 签名校验key
}

func (f *Feishu) formatURL() string {
	q := url.Values{}
	q.Add("type", string(TypeFeishu))
	q.Add("url", f.URL)
	q.Add("at", f.At)
	q.Add("signSecret", f.SignSecret)
	return fmt.Sprintf("http://%s?%s", alertProxyReceiverHost, q.Encode())
}

func (f *Feishu) ToReceiver(name string) v1alpha1.Receiver {
	u := f.formatURL()
	return v1alpha1.Receiver{
		Name: name,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL:          &u,
				SendResolved: utils.BoolPointer(f.SendResolved),
			},
		},
	}
}

func (f *Feishu) Check() error {
	if !strings.Contains(f.URL, "open.feishu.cn") {
		return fmt.Errorf("feishu robot url not valid")
	}
	return nil
}

func (f *Feishu) Test(alert prometheus.WebhookAlert) error {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(alert); err != nil {
		return err
	}
	resp, err := http.Post(f.formatURL(), "application/json", buf)
	if err != nil {
		return err
	}
	bts, _ := io.ReadAll(resp.Body)
	log.Info("test webhook success", "url", f.formatURL(), "resp", string(bts))
	return nil
}

func (f *Feishu) String() string {
	return f.formatURL()
}
