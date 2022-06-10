package install

import (
	"fmt"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestParsePluginsFrom(t *testing.T) {
	pluginpath := "../../../deploy/plugins"
	globalvalues := GlobalValues{}

	got, err := ParsePluginsFrom(pluginpath, globalvalues)
	if err != nil {
		t.Errorf("ParsePluginsFrom() error = %v", err)
		return
	}
	pluginscontent, _ := yaml.Marshal(got)
	fmt.Println(string(pluginscontent))
}
