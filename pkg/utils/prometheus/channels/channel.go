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
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type ChannelType string

const (
	TypeWebhook ChannelType = "webhook"
	TypeEmail   ChannelType = "email"
	TypeFeishu  ChannelType = "feishu"
)

var (
	KubegemsWebhookURL = fmt.Sprintf("https://kubegems-local-agent.%s:8041/alert", gems.NamespaceLocal)
)

type ChannelIf interface {
	ToReceiver(name string) v1alpha1.Receiver
	Check() error
	Test(alert prometheus.WebhookAlert) error
	String() string
}

type ChannelConfig struct {
	ChannelIf
}

type ChannelGetter func(id uint) (ChannelIf, error)

type ChannelMapper struct {
	M   map[uint]ChannelIf
	Err error
}

func (m *ChannelMapper) FindChannel(id uint) (ChannelIf, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	ret, ok := m.M[id]
	if !ok {
		return nil, fmt.Errorf("channel: %d not found", id)
	}
	return ret, nil
}

// Value return json value, implement driver.Valuer interface
func (m ChannelConfig) Value() (driver.Value, error) {
	bts, err := m.MarshalJSON()
	return string(bts), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *ChannelConfig) Scan(val interface{}) error {
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("failed to scan value:", val))
	}
	return m.UnmarshalJSON(ba)
}

// MarshalJSON to output non base64 encoded []byte
func (m ChannelConfig) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.ChannelIf)
}

// UnmarshalJSON to deserialize []byte
func (m *ChannelConfig) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		m.ChannelIf = nil
		return nil
	}
	tmp := struct {
		ChannelType `json:"channelType"`
	}{}
	if err := json.Unmarshal(b, &tmp); err != nil {
		return errors.Wrap(err, "unmarshal channelType")
	}
	switch tmp.ChannelType {
	case TypeWebhook:
		webhook := Webhook{}
		if err := json.Unmarshal(b, &webhook); err != nil {
			return errors.Wrap(err, "unmarshal webhook channel")
		}
		m.ChannelIf = &webhook
	case TypeEmail:
		email := Email{}
		if err := json.Unmarshal(b, &email); err != nil {
			return errors.Wrap(err, "unmarshal email channel")
		}
		m.ChannelIf = &email
	case TypeFeishu:
		feishu := Feishu{}
		if err := json.Unmarshal(b, &feishu); err != nil {
			return errors.Wrap(err, "unmarshal feishu channel")
		}
		m.ChannelIf = &feishu
	default:
		return fmt.Errorf("unknown channel type: %s", tmp.ChannelType)
	}
	return nil
}

// GormDataType gorm common data type
func (m ChannelConfig) GormDataType() string {
	return "channelConfig"
}

// GormDBDataType gorm db data type
func (ChannelConfig) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}
