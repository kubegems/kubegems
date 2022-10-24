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

package observe

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	v1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type Action int

const (
	Add Action = iota
	Update
	Delete
)

func GetBaseAlertmanagerConfig(namespace, name string) *v1alpha1.AlertmanagerConfig {
	return &v1alpha1.AlertmanagerConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.AlertmanagerConfigKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				gems.LabelAlertmanagerConfigName: name,
				gems.LabelAlertmanagerConfigType: prometheus.AlertTypeMonitor,
			},
		},
		Spec: v1alpha1.AlertmanagerConfigSpec{
			Route: &v1alpha1.Route{
				GroupBy:       []string{prometheus.AlertNamespaceLabel, prometheus.AlertNameLabel},
				GroupWait:     "30s",
				GroupInterval: "30s",
				Continue:      false,
				Receiver:      prometheus.NullReceiverName, // 默认发给空接收器，避免defaultReceiver收到不该收到的alert
			},
			Receivers:    []v1alpha1.Receiver{prometheus.NullReceiver, models.DefaultChannel.ToReceiver()},
			InhibitRules: []v1alpha1.InhibitRule{},
		},
	}
}

func GetBasePrometheusRule(namespace, name string) *monitoringv1.PrometheusRule {
	return &monitoringv1.PrometheusRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: monitoringv1.SchemeGroupVersion.String(),
			Kind:       monitoringv1.PrometheusRuleKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				gems.LabelPrometheusRuleName: name,
				gems.LabelPrometheusRuleType: prometheus.AlertTypeMonitor,
			},
		},
		Spec: monitoringv1.PrometheusRuleSpec{},
	}
}
