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
	"fmt"
	"net/url"
	"strings"

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type Feishu struct {
	ChannelType `json:"channelType"`
	URL         string `json:"url" binding:"required"` // feishu robot webhook url
	At          string `json:"at"`                     // 要@的用户id，所有人则是 all
	SignSecret  string `json:"signSecret"`             // 签名校验key
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
				URL: &u,
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
	return testAlertproxy(f.formatURL(), alert)
}

func (f *Feishu) String() string {
	return f.formatURL()
}
