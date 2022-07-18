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
	"time"
)

/*
ALTER TABLE environment_resources RENAME COLUMN cluster_name TO cluster;
ALTER TABLE environment_resources RENAME COLUMN tenant_name TO tenant;
ALTER TABLE environment_resources RENAME COLUMN project_name TO project;
ALTER TABLE environment_resources RENAME COLUMN environment_name TO environment;
*/

// EnvironmentResource Project资源使用清单
type EnvironmentResource struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `sql:"DEFAULT:'current_timestamp'"`

	Cluster     string
	Tenant      string
	Project     string
	Environment string

	MaxCPUUsageCore    float64
	MaxMemoryUsageByte float64
	MinCPUUsageCore    float64
	MinMemoryUsageByte float64
	AvgCPUUsageCore    float64
	AvgMemoryUsageByte float64
	NetworkReceiveByte float64
	NetworkSendByte    float64

	MaxPVCUsageByte float64
	MinPVCUsageByte float64
	AvgPVCUsageByte float64
}
