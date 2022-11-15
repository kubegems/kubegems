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

	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

// AliyunMsg 阿里云短信
type AliyunMsg struct {
	ChannelType     `json:"channelType"`
	AccessKeyId     string `json:"accessKeyId" binding:"required"`
	AccessKeySecret string `json:"accessKeySecret" binding:"required"`
	PhoneNumbers    string `json:"phoneNumbers" binding:"required"` // 电话号码，多个中间以","隔开
	SignName        string `json:"signName" binding:"required"`     // 签名
	TemplateCode    string `json:"templateCode" binding:"required"` // 模板
}

func (m *AliyunMsg) formatURL() string {
	q := url.Values{}
	q.Add("type", string(TypeAliyunMsg))
	q.Add("accessKeyId", m.AccessKeyId)
	q.Add("accessKeySecret", m.AccessKeySecret)
	q.Add("phoneNumbers", m.PhoneNumbers)
	q.Add("signName", m.SignName)
	q.Add("templateCode", m.TemplateCode)

	return fmt.Sprintf("http://%s?%s", alertProxyReceiverHost, q.Encode())
}

func (m *AliyunMsg) ToReceiver(name string) v1alpha1.Receiver {
	u := m.formatURL()
	return v1alpha1.Receiver{
		Name: name,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL: &u,
			},
		},
	}
}

func (m *AliyunMsg) Check() error {
	return utils.CheckStructFieldsEmpty(m)
}

func (m *AliyunMsg) Test(alert prometheus.WebhookAlert) error {
	return testAlertproxy(m.formatURL(), alert)
}
