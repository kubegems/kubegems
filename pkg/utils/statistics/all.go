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

package statistics

import (
	"context"

	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterStatistics struct {
	Version   string                    `json:"version"`
	Resources ClusterResourceStatistics `json:"resources"`
	Workloads ClusterWorkloadStatistics `json:"workloads"`
}

func GetClusterAllStatistics(ctx context.Context, cli client.Client, discovery discovery.DiscoveryInterface) *ClusterStatistics {
	return &ClusterStatistics{
		Version: func() string {
			sv, err := discovery.ServerVersion()
			if err != nil {
				return ""
			}
			return sv.String()
		}(),
		Resources: GetClusterResourceStatistics(ctx, cli),
		Workloads: GetWorkloadsStatistics(ctx, cli),
	}
}
