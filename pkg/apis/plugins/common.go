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

package plugins

const (
	AnnotationIgnoreOptions        = "bundle.kubegems.io/ignore-options"
	AnnotationIgnoreOptionOnUpdate = "OnUpdate"
	AnnotationIgnoreOptionOnDelete = "OnDelete"
)

const (
	// mark a helm chart as a kubegems plugin
	AnnotationIsPlugin = "plugins.kubegems.io/is-plugin"

	// plugin category
	// example: kubernetes/security,core/network
	AnnotationCategory = "plugins.kubegems.io/category"

	// health check target
	// example: deployment/*,statefulset/<name>,deployment/<prefix>*
	AnnotationHealthCheck = "plugins.kubegems.io/health-check"

	AnnotationRequired = "plugins.kubegems.io/required"

	// where the 'plugin' should install to
	AnnotationInstallNamespace = "plugins.kubegems.io/install-namespace"

	// ref values from configmap in "[namespace/]<name>" format,multiple split by ','
	// example: "kubegems/global,logging" .
	AnnotationValuesFrom = "plugins.kubegems.io/values-from"

	// description
	AnnotationPluginDescription = "plugins.kubegems.io/description"

	// required dependecies
	// example: foo > 1.0.0,bar = 1.2.0,kubegems ^ 1.20.0
	AnnotationRequirements = "plugins.kubegems.io/requirements"

	// specified which engine to render this plugin
	AnnotationRenderBy = "plugins.kubegems.io/render-by"
)

const (
	LabelIsPluginRepo = "plugins.kubegems.io/is-plugin-repo"
)

const (
	KubeGemsGlobalValuesConfigMapName = "kubegems-global-values"

	KubegemsChartInstaller = "kubegems-installer"
	KubegemsChartLocal     = "kubegems-local"
	KubegemsChartGlobal    = "global"

	KubeGemsNamespaceInstaller = "kubegems-installer"
	KubeGemsNamespaceLocal     = "kubegems-local"
)
