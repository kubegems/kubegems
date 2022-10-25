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

package models

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/prometheus/channels"
)

var (
	DefaultChannel = &AlertChannel{
		ID:   1,
		Name: channels.DefaultChannelName,
		ChannelConfig: channels.ChannelConfig{
			ChannelIf: &channels.Webhook{
				ChannelType:        channels.TypeWebhook,
				URL:                channels.KubegemsWebhookURL,
				InsecureSkipVerify: true,
			},
		},
	}
	DefaultReceiver = DefaultChannel.ChannelConfig.ChannelIf.ToReceiver(DefaultChannel.ReceiverName())
)

// AlertChannel
type AlertChannel struct {
	ID            uint                   `gorm:"primarykey" json:"id"`
	Name          string                 `gorm:"type:varchar(50)" binding:"required" json:"name"`
	ChannelConfig channels.ChannelConfig `json:"channelConfig"`

	TenantID *uint   `json:"tenantID"` // 若为null，则表示系统预置
	Tenant   *Tenant `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;" json:"tenant,omitempty"`

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

func (c *AlertChannel) ReceiverName() string {
	return fmt.Sprintf("%s-id-%d", c.Name, c.ID)
}

var receiverNameReg = regexp.MustCompile("(.*)-id-(.*)")

func ChannelIDNameByReceiverName(recName string) (string, uint) {
	substrs := receiverNameReg.FindStringSubmatch(recName)
	if len(substrs) == 3 {
		name := substrs[1]
		id, err := strconv.Atoi(substrs[2])
		if err != nil {
			log.Errorf("channel id %s not valid", id)
		}
		return name, uint(id)
	}
	log.Errorf("receiver name %s not valid", recName)
	return "", 0
}

func NewChannnelMappler(db *gorm.DB) *channels.ChannelMapper {
	allch := []AlertChannel{}
	if err := db.Find(&allch).Error; err != nil {
		return &channels.ChannelMapper{Err: err}
	}
	ret := &channels.ChannelMapper{
		M: make(map[uint]channels.ChannelIf),
	}
	for _, v := range allch {
		ret.M[v.ID] = v.ChannelConfig.ChannelIf
	}
	return ret
}
