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

// AliyunVoice 阿里云语音
type AliyunVoice struct {
	BaseChannel     `json:",inline"`
	AccessKeyId     string `json:"accessKeyId" binding:"required"`
	AccessKeySecret string `json:"accessKeySecret" binding:"required"`
	CallNumber      string `json:"callNumber" binding:"required"` // 电话号码，只支持单个
	TtsCode         string `json:"ttsCode" binding:"required"`    // 模板
}

func (v *AliyunVoice) formatURL() string {
	q := url.Values{}
	q.Add("type", string(TypeAliyunVoice))
	q.Add("accessKeyId", v.AccessKeyId)
	q.Add("accessKeySecret", v.AccessKeySecret)
	q.Add("callNumber", v.CallNumber)
	q.Add("ttsCode", v.TtsCode)

	return fmt.Sprintf("http://%s?%s", alertProxyReceiverHost, q.Encode())
}

func (v *AliyunVoice) ToReceiver(name string) v1alpha1.Receiver {
	u := v.formatURL()
	return v1alpha1.Receiver{
		Name: name,
		WebhookConfigs: []v1alpha1.WebhookConfig{
			{
				URL: &u,
			},
		},
	}
}

func (v *AliyunVoice) Check() error {
	return utils.CheckStructFieldsEmpty(v)
}

func (v *AliyunVoice) Test(alert prometheus.WebhookAlert) error {
	return testAlertproxy(v.formatURL(), alert)
}

func (v *AliyunVoice) String() string {
	return v.formatURL()
}
