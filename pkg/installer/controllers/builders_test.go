package controllers

import (
	"context"
	"io/ioutil"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	pluginv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"sigs.k8s.io/yaml"
)

func TestTemplatesBuildPlugin(t *testing.T) {
	type args struct {
		ctx    context.Context
		plugin *Plugin
	}
	tests := []struct {
		name       string
		args       args
		want       []*unstructured.Unstructured
		ignoreWant bool
		wantErr    bool
	}{
		{
			name: "test build kubegem-local-stack",
			args: args{
				ctx: context.Background(),
				plugin: func(t *testing.T) *Plugin {
					plugindefinition := "../../../deploy/plugins/kubegems-local-stack.yaml"
					plugindpath := "../../../deploy/plugins/kubegems-local-stack"
					content, err := ioutil.ReadFile(plugindefinition)
					if err != nil {
						t.Errorf("read plugin definition: %v", err)
					}
					plugin := &pluginv1beta1.Plugin{}
					if err := yaml.Unmarshal(content, plugin); err != nil {
						t.Errorf("unmarshal plugin definition: %v", err)
					}
					plg := PluginFromPlugin(plugin)
					plg.Path = plugindpath
					return plg
				}(t),
			},
			ignoreWant: true,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := (&Templater{}).Template(tt.args.ctx, tt.args.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplatesBuildPlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.ignoreWant && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TemplatesBuildPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}
