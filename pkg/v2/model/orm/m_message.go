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

package orm

import (
	"time"

	"gorm.io/datatypes"
)

// +gen type:object pkcolume:id pkfield:ID
type UserMessageStatus struct {
	ID        uint
	UserID    uint
	User      *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	MessageID uint
	Message   *Message `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	IsRead    bool
}

// +gen type:object pkcolume:id pkfield:ID
type Message struct {
	ID          uint   `gorm:"primarykey"`
	MessageType string `gorm:"type:varchar(50);"`
	Title       string `gorm:"type:varchar(255);"`
	Content     datatypes.JSON
	CreatedAt   *time.Time        `gorm:"index" sql:"DEFAULT:'current_timestamp'"`
	ToUsers     map[uint]struct{} `gorm:"-" json:"-"`
}
