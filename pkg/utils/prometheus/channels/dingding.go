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
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type Dingding struct {
	BaseChannel `json:",inline"`
	URL         string `json:"url" binding:"required"` // Dingding robot webhook url
	AtMobiles   string `json:"atMobiles"`              // 要@的用户手机号
	SignSecret  string `json:"signSecret"`             // 签名校验key
}

func (f *Dingding) formatURL() string {
	q := url.Values{}
	q.Add("type", string(TypeDingding))
	q.Add("url", f.URL)
	q.Add("atMobiles", f.AtMobiles)
	q.Add("signSecret", f.SignSecret)
	return fmt.Sprintf("http://%s?%s", alertProxyReceiverHost, q.Encode())
}

func (f *Dingding) ToReceiver(name string) v1alpha1.Receiver {
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

func (f *Dingding) Check() error {
	if !strings.Contains(f.URL, "oapi.dingtalk.com") {
		return fmt.Errorf("Dingding robot url not valid")
	}
	return nil
}

func (f *Dingding) Test(alert prometheus.WebhookAlert) error {
	return testAlertproxy(f.formatURL(), alert)
}

func (f *Dingding) String() string {
	return f.formatURL()
}
