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

package forms

// +genform object:Tenant
type TenantCommon struct {
	BaseForm
	ID   uint   `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// +genform object:Tenant
type TenantDetail struct {
	BaseForm
	ID       uint          `json:"id,omitempty"`
	Name     string        `json:"name,omitempty"`
	Remark   string        `json:"remark,omitempty"`
	IsActive bool          `json:"isActive,omitempty"`
	Users    []*UserCommon `json:"users,omitempty"`
}

// +genform object:TenantUserRel
type TenantUserRelCommon struct {
	BaseForm
	ID       uint          `json:"id,omitempty"`
	Tenant   *TenantCommon `json:"tenant,omitempty"`
	TenantID uint          `json:"tenantID,omitempty"`
	User     *UserCommon   `json:"user,omitempty"`
	UserID   uint          `json:"userID,omitempty"`
	Role     string        `json:"role,omitempty"`
}

type TenantUserCreateModifyForm struct {
	BaseForm
	Tenant string `json:"tenant" validate:"required"`
	User   string `json:"user" validate:"required"`
	Role   string `json:"role" validate:"required"`
}
