package main

import (
	"io"
	"os"

	"github.com/ghodss/yaml"
	"kubegems.io/pkg/utils/prometheus"
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
			AMConfig: prometheus.GetBaseAlertmanagerConfig(prometheus.GlobalAlertNamespace, prometheus.MonitorAlertmanagerConfigName),
		},
		PrometheusRule: prometheus.GetBasePrometheusRule(prometheus.GlobalAlertNamespace),
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

	amout, _ := yaml.Marshal(raw.Base.AMConfig)
	prout, _ := yaml.Marshal(raw.PrometheusRule)
	os.WriteFile("../installer/roles/installer/files/alertmanager/config.yaml", amout, 0644)
	os.WriteFile("../installer/roles/installer/files/prometheus/alerting.rule.yaml", prout, 0644)
}
