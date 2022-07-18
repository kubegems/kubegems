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

import "time"

// User 用户表
// +gen type:object pkcolume:id pkfield:ID preloads:SystemRole
type User struct {
	ID           uint       `gorm:"primarykey"`
	Name         string     `gorm:"type:varchar(50);uniqueIndex"`
	Email        string     `gorm:"type:varchar(50)"`
	Phone        string     `gorm:"type:varchar(255)"`
	Password     string     `gorm:"type:varchar(255)"`
	Source       string     `gorm:"type:varchar(255)"`
	IsActive     *bool      `sql:"DEFAULT:true"`
	CreatedAt    *time.Time `sql:"DEFAULT:'current_timestamp'"`
	LastLoginAt  *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Tenants      []*Tenant  `gorm:"many2many:tenant_user_rels;"`
	SystemRole   *SystemRole
	SystemRoleID uint
	Role         string `sql:"-"`
}
