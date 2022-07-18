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

// +gen type:object pkcolume:id pkfield:ID
type OpenAPP struct {
	Name      string `gorm:"unique"`
	ID        uint
	AppID     string
	AppSecret string
	// 系统权限范围,空则表示什么操作都不行,默认是ReadWorkload
	PermScopes string `sql:"DEFAULT:'ReadWorkload'"`
	// 可操作租户范围，通过id列表表示，逗号分隔，可以用通配符 *，表示所有, 默认*
	TenantScope string `sql:"DEFAULT:'*'"`
	// 访问频率限制，空则表示不限制,表示每分钟可以访问的次数，默认30
	RequestLimiter int `sql:"DEFAULT:30"`
}
