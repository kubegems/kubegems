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

package tenanthandler

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"kubegems.io/kubegems/pkg/utils/statistics"
)

func TestCheckOverSold(t *testing.T) {
	type args struct {
		clusterstatistics statistics.ClusterResourceStatistics
		oversoldRate      map[v1.ResourceName]float32
		before            v1.ResourceList
		after             v1.ResourceList
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				clusterstatistics: statistics.ClusterResourceStatistics{
					Capacity:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("2")},
					TenantAllocated: v1.ResourceList{v1.ResourceCPU: resource.MustParse("1")},
				},
				oversoldRate: map[v1.ResourceName]float32{v1.ResourceCPU: 1},
				before:       v1.ResourceList{v1.ResourceCPU: resource.MustParse("1")},
				after:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("2")},
			},
		},
		{
			name: "overflow",
			args: args{
				clusterstatistics: statistics.ClusterResourceStatistics{
					Capacity:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("4")},
					TenantAllocated: v1.ResourceList{v1.ResourceCPU: resource.MustParse("5")},
				},
				oversoldRate: map[v1.ResourceName]float32{v1.ResourceCPU: 1.5},
				before:       v1.ResourceList{v1.ResourceCPU: resource.MustParse("2")},
				after:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("4")},
			},
			wantErr: true,
		},
		{
			name: "not allocated",
			args: args{
				clusterstatistics: statistics.ClusterResourceStatistics{
					Capacity: v1.ResourceList{v1.ResourceCPU: resource.MustParse("4")},
				},
				oversoldRate: map[v1.ResourceName]float32{v1.ResourceCPU: 1},
				after:        v1.ResourceList{v1.ResourceCPU: resource.MustParse("4")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckOverSold(tt.args.clusterstatistics, tt.args.oversoldRate, tt.args.before, tt.args.after); (err != nil) != tt.wantErr {
				t.Errorf("CheckOverSold() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
