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

import (
	"encoding/json"
	"time"
)

// +genform object:OpenAPP
type OpenAPPCommon struct {
	BaseForm
	Name           string `json:"name,omitempty"`
	ID             uint   `json:"id,omitempty"`
	AppID          string `json:"appID,omitempty"`
	PermScopes     string `json:"permScopes,omitempty"`
	TenantScope    string `json:"tenantScope,omitempty"`
	RequestLimiter int    `json:"requestLimiter,omitempty"`
}

// +genform object:OpenAPP
type OpenAPPDetail struct {
	BaseForm
	Name           string `json:"name,omitempty"`
	ID             uint   `json:"id,omitempty"`
	AppID          string `json:"appID,omitempty"`
	AppSecret      string `json:"appSecret,omitempty"`
	PermScopes     string `json:"permScopes,omitempty"`
	TenantScope    string `json:"tenantScope,omitempty"`
	RequestLimiter int    `json:"requestLimiter,omitempty"`
}

func (u *OpenAPPDetail) GetID() uint {
	return u.ID
}

func (u *OpenAPPDetail) SetLastLogin(t *time.Time) {
}

func (u *OpenAPPDetail) GetSystemRoleID() uint {
	return 0
}

func (u *OpenAPPDetail) GetUsername() string {
	return u.Name
}

func (u *OpenAPPDetail) GetUserKind() string {
	return "app"
}

func (u *OpenAPPDetail) GetEmail() string {
	return ""
}

func (u *OpenAPPDetail) GetSource() string {
	return "app"
}

func (i *OpenAPPDetail) MarshalBinary() ([]byte, error) {
	return json.Marshal(i)
}

func (i *OpenAPPDetail) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, i)
}
