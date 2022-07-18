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

package helm

import (
	"context"
	"testing"
)

func TestSyncChartsToChartmuseum(t *testing.T) {
	type args struct {
		ctx              context.Context
		remote           RepositoryConfig
		localChartMuseum RepositoryConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "",
			args: args{
				ctx: context.Background(),
				remote: RepositoryConfig{
					// Name: "bitnami",
					// URL:  "https://charts.bitnami.com/bitnami",
					Name: "apisix",
					URL:  "https://charts.apiseven.com",
				},
				localChartMuseum: RepositoryConfig{
					Name: "",
					URL:  "http://172.16.23.121:31459",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SyncChartsToChartmuseum(tt.args.ctx, tt.args.remote, tt.args.localChartMuseum); (err != nil) != tt.wantErr {
				t.Errorf("SyncChartsToChartmuseum() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
