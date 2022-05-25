package main

import (
	"bytes"
	"io"
	"os"
	"regexp"

	"github.com/ghodss/yaml"
	"kubegems.io/pkg/apis/gems"
	"kubegems.io/pkg/utils/prometheus"
)

const (
	nstmpl = "{{ .Values.monitoring.namespace }}"
)

var (
	agentURL = `https://kubegems-local-agent.{{ index .Values "kubegems-local" "namespace" }}:8041/alert`
)

func main() {
	alerts := []prometheus.MonitorAlertRule{}
	file, err := os.Open("scripts/generate-system-alert/system-alert.yaml")
	if err != nil {
		panic(err)
	}
	bts, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(bts, &alerts); err != nil {
		panic(err)
	}

	raw := &prometheus.RawMonitorAlertResource{
		Base: &prometheus.BaseAlertResource{
			AMConfig: prometheus.GetBaseAlertmanagerConfig(gems.NamespaceMonitor, prometheus.MonitorAlertmanagerConfigName),
		},
		PrometheusRule: prometheus.GetBasePrometheusRule(gems.NamespaceMonitor),
		MonitorOptions: prometheus.DefaultMonitorOptions(),
	}

	for _, alert := range alerts {
		if err := alert.CheckAndModify(raw.MonitorOptions); err != nil {
			panic(err)
		}
		if err := raw.ModifyAlertRule(alert, prometheus.Add); err != nil {
			panic(err)
		}
	}

	raw.Base.AMConfig.Spec.Receivers[1].WebhookConfigs[0].URL = &agentURL

	os.WriteFile("deploy/plugins/kubegems-local-stack/templates/monitoring/kubegems-default-monitor-amconfig.yaml", getOutput(raw.Base.AMConfig), 0644)
	os.WriteFile("deploy/plugins/kubegems-local-stack/templates/monitoring/kubegems-default-alert-rule.yaml", getOutput(raw.PrometheusRule), 0644)
}

var reg = regexp.MustCompile("{{ %")

func getOutput(obj interface{}) []byte {
	bts, _ := yaml.Marshal(obj)
	// 对不需要替换的'{{`', '}}'转义，https://stackoverflow.com/questions/17641887/how-do-i-escape-and-delimiters-in-go-templates

	bts = bytes.ReplaceAll(bts, []byte(":{{"), []byte(`:{{"{{"}}`))
	// bts = bytes.ReplaceAll(bts, []byte("}}]"), []byte(`{{"}}"}}]`))
	bts = bytes.ReplaceAll(bts, []byte("{{ $value"), []byte(`{{"{{ $value"}}`))
	buf := bytes.NewBuffer([]byte("{{- if .Values.monitoring.enabled -}}\n"))
	buf.Write(bytes.ReplaceAll(bts, []byte(gems.NamespaceMonitor), []byte(nstmpl)))
	buf.Write([]byte("{{- end }}"))

	return buf.Bytes()
}
