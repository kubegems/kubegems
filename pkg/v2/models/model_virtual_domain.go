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

import "time"

const (
	ResVirtualDomain = "virtualDomain"
)

/*
ALTER TABLE virtual_domains RENAME COLUMN virtual_domain_name TO name;
*/

type VirtualDomain struct {
	ID   uint   `gorm:"primarykey"`
	Name string `gorm:"type:varchar(50);uniqueIndex"`

	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`
	UpdatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	IsActive  bool // 是否激活
	CreatedBy string
}
