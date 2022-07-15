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
	"strconv"
	"time"

	"gorm.io/datatypes"
	"kubegems.io/kubegems/pkg/utils"
)

type UserMessageStatus struct {
	ID        uint
	UserID    uint
	User      *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	MessageID *uint
	Message   *Message `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	IsRead    bool     `gorm:"index"`

	AlertMessageID *uint
	AlertMessage   *AlertMessage `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Message struct {
	ID          uint   `gorm:"primarykey"`
	MessageType string `gorm:"type:varchar(50);"`
	Title       string `gorm:"type:varchar(255);"`
	Content     datatypes.JSON
	CreatedAt   time.Time         `gorm:"index" sql:"DEFAULT:'current_timestamp'"`
	ToUsers     map[uint]struct{} `gorm:"-" json:"-"`

	IsRead bool `gorm:"-"` // 给前端用，不入库
}

func (status *UserMessageStatus) ColumnSlice() []string {
	return []string{"id", "user_id", "message_id", "is_read", "alert_message_id"}
}

func (status *UserMessageStatus) ValueSlice() []string {
	return []string{
		strconv.Itoa(int(status.ID)),
		strconv.Itoa(int(status.UserID)),
		utils.UintToStr(status.MessageID),
		utils.BoolToString(status.IsRead),
		utils.UintToStr(status.AlertMessageID),
	}
}

func (msg *Message) ColumnSlice() []string {
	return []string{"id", "message_type", "title", "content", "created_at"}
}

func (msg *Message) ValueSlice() []string {
	return []string{
		strconv.Itoa(int(msg.ID)),
		msg.MessageType,
		msg.Title,
		string(msg.Content),
		utils.FormatMysqlDumpTime(&msg.CreatedAt), // mysql datetime 格式
	}
}
