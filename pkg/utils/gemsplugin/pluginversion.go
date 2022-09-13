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

package gemsplugin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
)

type PluginVersion struct {
	Name             string                      `json:"name,omitempty"`
	Namespace        string                      `json:"namespace,omitempty"`
	Enabled          bool                        `json:"enabled,omitempty"`
	InstallNamespace string                      `json:"installNamespace,omitempty"`
	Kind             pluginsv1beta1.BundleKind   `json:"kind,omitempty"`
	Description      string                      `json:"description,omitempty"`
	HelathCheck      string                      `json:"helathCheck,omitempty"`
	MainCategory     string                      `json:"mainCategory,omitempty"`
	Category         string                      `json:"category,omitempty"`
	Repository       string                      `json:"repository,omitempty"`
	Version          string                      `json:"version,omitempty"`
	Healthy          bool                        `json:"healthy,omitempty"`
	Required         bool                        `json:"required,omitempty"`
	Requirements     string                      `json:"requirements,omitempty"` // dependecies requirements
	Message          string                      `json:"message,omitempty"`
	Values           pluginsv1beta1.Values       `json:"values,omitempty"`
	Schema           string                      `json:"schema,omitempty"`
	ValuesFrom       []pluginsv1beta1.ValuesFrom `json:"valuesFrom,omitempty"`
}

func (pv PluginVersion) ToPlugin() *pluginsv1beta1.Plugin {
	plugininfo, _ := json.Marshal(pv)
	annotations := map[string]string{
		pluginscommon.AnnotationPluginInfo: string(plugininfo),
	}
	if pv.Kind == "" {
		pv.Kind = pluginsv1beta1.BundleKindTemplate // prefer use template with plugin
	}
	return &pluginsv1beta1.Plugin{
		ObjectMeta: v1.ObjectMeta{
			Name:        pv.Name,
			Namespace:   pv.Namespace,
			Annotations: annotations,
		},
		Spec: pluginsv1beta1.PluginSpec{
			Kind:       pv.Kind,
			URL:        pv.Repository,
			Chart:      pv.Name,
			Version:    pv.Version,
			Values:     pv.Values,
			ValuesFrom: pv.ValuesFrom,
		},
	}
}

func PluginVersionFrom(plugin *pluginsv1beta1.Plugin) PluginVersion {
	annotations := plugin.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	pv := PluginVersion{}
	_ = json.Unmarshal([]byte(annotations[pluginscommon.AnnotationPluginInfo]), &pv)

	pv.Name = plugin.Name
	pv.Namespace = plugin.Namespace
	pv.InstallNamespace = plugin.Spec.InstallNamespace
	pv.Version = plugin.Spec.Version
	pv.Enabled = !plugin.Spec.Disabled
	pv.Repository = plugin.Spec.URL
	pv.Message = plugin.Status.Message
	if pv.Version == "" {
		pv.Version = plugin.Status.Version
	}
	pv.ValuesFrom = plugin.Spec.ValuesFrom
	if plugin.Status.Phase == pluginsv1beta1.PhaseInstalled {
		pv.Healthy = true
	}
	return pv
}

func PluginVersionFromRepoChartVersion(repo string, cv *repo.ChartVersion) PluginVersion {
	annotations := cv.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	required, _ := strconv.ParseBool(annotations[pluginscommon.AnnotationRequired])

	maincate, cate := "other", "unknow"
	categories := strings.Split(annotations[pluginscommon.AnnotationCategory], "/")
	if len(categories) == 1 {
		cate = categories[0]
	} else if len(categories) > 1 {
		maincate, cate = categories[0], categories[1]
	}

	valsFrom := []pluginsv1beta1.ValuesFrom{
		// always inject the global values reference in plugin
		{
			Kind:     pluginsv1beta1.ValuesFromKindConfigmap,
			Name:     pluginscommon.KubegemsChartGlobal,
			Prefix:   pluginscommon.KubegemsChartGlobal + ".",
			Optional: true,
		},
	}
	for _, val := range strings.Split(annotations[pluginscommon.AnnotationValuesFrom], ",") {
		if val == "" || val == pluginscommon.KubegemsChartGlobal {
			continue
		}
		namespace, name := "", val
		if splits := strings.Split(val, "/"); len(splits) > 1 {
			namespace, name = splits[0], splits[1]
		}
		valsFrom = append(valsFrom, pluginsv1beta1.ValuesFrom{
			Kind:      pluginsv1beta1.ValuesFromKindConfigmap,
			Name:      name,
			Namespace: namespace,
			Prefix:    name + ".",
			Optional:  true,
		})
	}

	return PluginVersion{
		Name:             cv.Name,
		Kind:             pluginsv1beta1.BundleKindTemplate,
		Repository:       repo,
		InstallNamespace: annotations[pluginscommon.AnnotationInstallNamespace],
		Version:          cv.Version,
		Description:      cv.Description,
		MainCategory:     maincate,
		Category:         cate,
		ValuesFrom:       valsFrom,
		Required:         required,
		Requirements:     annotations[pluginscommon.AnnotationRequirements],
		HelathCheck:      annotations[pluginscommon.AnnotationHealthCheck],
	}
}

func IsPluginChart(cv *repo.ChartVersion) bool {
	annotations := cv.Annotations
	if annotations == nil {
		return false
	}
	b, _ := strconv.ParseBool(annotations[pluginscommon.AnnotationIsPlugin])
	return b
}

func FindUpgradeable(availables []PluginVersion, installed map[string]PluginVersion) *PluginVersion {
	for _, available := range availables {
		if CheckDependecies(available.Requirements, installed) == nil {
			return &available
		}
	}
	return nil
}

type ErrorList []error

func (list ErrorList) Error() string {
	msg := ""
	for _, item := range list {
		msg += item.Error() + ";"
	}
	return msg
}

func CheckDependecies(requirements string, installs map[string]PluginVersion) error {
	reqs := ParseRequirements(requirements)
	var errs ErrorList
	for _, req := range reqs {
		constraint, err := semver.NewConstraint(req.Constraint)
		if err != nil {
			continue
		}
		installed, ok := installs[req.Name]
		if !ok {
			errs = append(errs, fmt.Errorf("%s not installed,require: %s", req.Name, req.Constraint))
			continue
		}
		ver, err := semver.NewVersion(installed.Version)
		if err != nil {
			continue
		}
		if !constraint.Check(ver) {
			errs = append(errs, fmt.Errorf("%s not meet,require: %s", req.Name, req.Constraint))
		}
	}
	if len(errs) != 0 {
		return errs
	}
	return nil
}

type Requirement struct {
	Name       string
	Constraint string
}

// ParseRequirements
func ParseRequirements(str string) []Requirement {
	requirements := []Requirement{}
	// nolint: gomnd
	for _, req := range strings.Split(str, ",") {
		if req == "" {
			continue
		}
		splites := strings.SplitN(req, " ", 2)
		switch len(splites) {
		case 1:
			requirements = append(requirements, Requirement{Name: splites[0]})
		case 2:
			requirements = append(requirements, Requirement{Name: splites[0], Constraint: splites[1]})
		default:
			continue
		}
	}
	return requirements
}
