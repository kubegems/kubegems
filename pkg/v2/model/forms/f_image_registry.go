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

import "time"

// +genform object:Registry
type RegistryCommon struct {
	BaseForm
	ID         uint           `json:"id,omitempty"`
	Name       string         `json:"registryName,omitempty"`
	Address    string         `json:"address,omitempty"`
	UpdateTime *time.Time     `json:"updateTime,omitempty"`
	Creator    *UserCommon    `json:"creator,omitempty"`
	CreatorID  uint           `json:"creatorID,omitempty"`
	Project    *ProjectCommon `json:"project,omitempty"`
	ProjectID  uint           `json:"projectID,omitempty"`
	IsDefault  bool           `json:"isDefault,omitempty"`
}

// +genform object:Registry
type RegistryDetail struct {
	BaseForm
	ID         uint           `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Address    string         `json:"address,omitempty"`
	Username   string         `json:"username,omitempty"`
	Password   string         `json:"password,omitempty"`
	Creator    *UserCommon    `json:"creator,omitempty"`
	UpdateTime *time.Time     `json:"updateTime,omitempty"`
	CreatorID  uint           `json:"creatorID,omitempty"`
	Project    *ProjectCommon `json:"project,omitempty"`
	ProjectID  uint           `json:"projectID,omitempty"`
	IsDefault  bool           `json:"isDefault,omitempty"`
}
