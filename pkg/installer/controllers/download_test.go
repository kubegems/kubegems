package controllers

import (
	"context"
	"os"
	"testing"
)

func TestDownload(t *testing.T) {
	pwd, _ := os.Getwd()
	_ = pwd

	type args struct {
		ctx    context.Context
		plugin DownloadRepo
		into   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// {
		// 	name: "git",
		// 	args: args{
		// 		ctx: context.Background(),
		// 		plugin: DownloadRepo{
		// 			URI:     "https://github.com/rancher/local-path-provisioner.git",
		// 			SubPath: "deploy/chart",
		// 			Version: "v0.0.20",
		// 		},
		// 		into: "download/git",
		// 	},
		// },
		// {
		// 	name: "zip",
		// 	args: args{
		// 		ctx: context.Background(),
		// 		plugin: DownloadRepo{
		// 			URI:     "https://github.com/rancher/local-path-provisioner/archive/refs/heads/master.zip",
		// 			SubPath: "local-path-provisioner-master/deploy/chart/local-path-provisioner",
		// 		},
		// 		into: "download/zip",
		// 	},
		// },
		// {
		// 	name: "file",
		// 	args: args{
		// 		ctx: context.Background(),
		// 		plugin: DownloadRepo{
		// 			URI:     "file://" + pwd,
		// 			SubPath: "testdata/helm-test",
		// 		},
		// 		into: "download/file",
		// 	},
		// },
		{
			name: "helm",
			args: args{
				ctx: context.Background(),
				plugin: DownloadRepo{
					URI:     "https://charts.bitnami.com/bitnami",
					SubPath: "nginx-ingress-controller",
					// Version: "v1.0.0",
				},
				into: "download/helm",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Download(tt.args.ctx, tt.args.plugin, tt.args.into)
			if (err != nil) != tt.wantErr {
				t.Errorf("Download() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
