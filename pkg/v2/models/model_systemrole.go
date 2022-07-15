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

const (
	SystemRoleAdmin    = "sysadmin"
	SystemRoleOrdinary = "ordinary"

	ResSystemRole = "systemrole"
)

/*
ALTER TABLE system_roles RENAME COLUMN role_name TO name
ALTER TABLE system_roles RENAME COLUMN role_code TO code
*/

type SystemRole struct {
	ID    uint `gorm:"primarykey"`
	Name  string
	Code  string `gorm:"type:varchar(30)" binding:"required,eq=sysadmin|eq=normal"`
	Users []*User
}
