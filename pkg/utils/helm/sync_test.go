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
