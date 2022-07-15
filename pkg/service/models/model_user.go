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
	"encoding/json"
	"time"
)

const (
	ResUser = "user"
)

// User 用户表
type User struct {
	ID uint `gorm:"primarykey"`
	// 用户名
	Username string `gorm:"type:varchar(50);uniqueIndex" binding:"required"`
	// 邮箱
	Email string `gorm:"type:varchar(50)" binding:"required"`
	// 电话
	Phone    string `gorm:"type:varchar(255)"`
	Password string `gorm:"type:varchar(255)" json:"-"`
	// 是否激活
	IsActive *bool `sql:"DEFAULT:true"`
	// 加入时间
	CreatedAt *time.Time `sql:"DEFAULT:'current_timestamp'"`
	// 最后登录时间
	LastLoginAt *time.Time `sql:"DEFAULT:'current_timestamp'"`

	Source       string    `gorm:"type:varchar(50)"`
	SourceVendor string    `gorm:"type:varchar(50)"`
	Tenants      []*Tenant `gorm:"many2many:tenant_user_rels;"`
	SystemRole   *SystemRole
	SystemRoleID uint

	// 角色，不同关联对象下表示的角色不同, 用来做join查询的时候处理角色字段的(请勿删除)
	Role string `sql:"-" json:",omitempty"`
}

type UserSel struct {
	ID       uint
	Username string
	Email    string
}

// implement redis
func (u User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

func (u *User) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &u)
}

type UserCreate struct {
	ID       uint   `json:"id"`
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"min=8"`
}

func (u *User) GetID() uint {
	return u.ID
}
func (u *User) GetSystemRoleID() uint {
	return u.SystemRoleID
}
func (u *User) GetUsername() string {
	return u.Username

}
func (u *User) GetUserKind() string {
	return ""
}
func (u *User) GetMail() string {
	return u.Email
}
func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetSource() string {
	return u.Source
}
func (u *User) SetLastLogin(t *time.Time) {
	u.LastLoginAt = t
}
