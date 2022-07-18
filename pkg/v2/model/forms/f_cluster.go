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

import "gorm.io/datatypes"

// +genform object:Cluster
type ClusterCommon struct {
	BaseForm
	ID        uint   `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Primary   bool   `json:"primary,omitempty"`
	APIServer string `json:"apiServer,omitempty"`
	Version   string `json:"version,omitempty"`
	Runtime   string `json:"runtime,omitempty"`
}

// +genform object:Cluster
type ClusterDetail struct {
	BaseForm
	ID                   uint                 `json:"id,omitempty"`
	Name                 string               `json:"name,omitempty"`
	APIServer            string               `json:"apiServer,omitempty"`
	KubeConfig           datatypes.JSON       `json:"kubeConfig,omitempty"`
	Version              string               `json:"version,omitempty"`
	AgentAddr            string               `json:"agentAddr,omitempty"`
	AgentCA              string               `json:"agentCA,omitempty"`
	AgentCert            string               `json:"agentCert,omitempty"`
	AgentKey             string               `json:"agentKey,omitempty"`
	Mode                 string               `json:"mode,omitempty"`
	Runtime              string               `json:"runtime,omitempty"`
	Primary              bool                 `json:"primary,omitempty"`
	OversoldConfig       datatypes.JSON       `json:"oversoldConfig,omitempty"`
	Environments         []*EnvironmentCommon `json:"environments,omitempty"`
	ClusterResourceQuota datatypes.JSON       `json:"clusterResourceQuota,omitempty"`
}
