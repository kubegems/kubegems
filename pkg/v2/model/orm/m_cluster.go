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

import "gorm.io/datatypes"

// +gen type:object kind:cluster pkcolume:id pkfield:ID preloads:TenantResourceQuotas
type Cluster struct {
	ID         uint   `gorm:"primarykey"`
	Name       string `gorm:"type:varchar(50);uniqueIndex"`
	APIServer  string `gorm:"type:varchar(250);uniqueIndex"`
	KubeConfig datatypes.JSON

	Version   string
	AgentAddr string
	AgentCA   string
	AgentCert string
	AgentKey  string
	Mode      string
	Runtime   string // docker or containerd
	Primary   bool   // is primary cluster

	OversoldConfig       datatypes.JSON // cluster oversold configuration
	Environments         []*Environment
	TenantResourceQuotas []*TenantResourceQuota
	ClusterResourceQuota datatypes.JSON
}
