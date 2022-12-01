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

package gems

const (
	LabelTenant      = GroupName + "/tenant"
	LabelProject     = GroupName + "/project"
	LabelEnvironment = GroupName + "/environment"
	LabelApplication = GroupName + "/application"
	LabelZone        = GroupName + "/zone"
	LabelPlugins     = GroupName + "/plugins"

	NamespaceSystem    = "kubegems"
	NamespaceLocal     = "kubegems-local"
	NamespaceEdge      = "kubegems-edge"
	NamespaceInstaller = "kubegems-installer"
	NamespaceMonitor   = "kubegems-monitoring"
	NamespaceLogging   = "kubegems-logging"
	NamespaceGateway   = "kubegems-gateway"
	NamespaceEventer   = "kubegems-eventer"
	NamespaceObserve   = "observability"
)

var CommonLabels = []string{
	LabelTenant,
	LabelProject,
	LabelEnvironment,
}

const (
	FinalizerNamespace     = "finalizer." + GroupName + "/namespace"
	FinalizerResourceQuota = "finalizer." + GroupName + "/resourcequota"
	FinalizerGateway       = "finalizer." + GroupName + "/gateway"
	FinalizerNetworkPolicy = "finalizer." + GroupName + "/networkpolicy"
	FinalizerLimitrange    = "finalizer." + GroupName + "/limitrange"
	FinalizerEnvironment   = "finalizer." + GroupName + "/environment"
)

const (
	LabelMonitorCollector = GroupName + "/monitoring"
	LabelLogCollector     = GroupName + "/logging"

	LabelAlertmanagerConfigName = "alertmanagerconfig.kubegems.io/name"
	LabelAlertmanagerConfigType = "alertmanagerconfig.kubegems.io/type"

	LabelPrometheusRuleName = "prometheusrule.kubegems.io/name"
	LabelPrometheusRuleType = "prometheusrule.kubegems.io/type"

	LabelGatewayType = "gateway.kubegems.io/type" // ingress-nginx

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)
