package controllers

import (
	"context"
	"testing"
)

func TestDownload(t *testing.T) {
	type args struct {
		ctx    context.Context
		plugin Plugin
		tag    string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "git",
			args: args{
				ctx: context.Background(),
				plugin: Plugin{
					Repo:    "https://github.com/rancher/local-path-provisioner.git",
					Path:    "deploy/chart",
					Version: "v0.0.20",
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Download(tt.args.ctx, tt.args.plugin.Repo, tt.args.plugin.Version, tt.args.plugin.Path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Download() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Download() = %v, want %v", got, tt.want)
			}
		})
	}
}
