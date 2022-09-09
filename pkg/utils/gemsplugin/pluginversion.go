package gemsplugin

import (
	"encoding/json"
	"strconv"
	"strings"

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

	kind := pluginsv1beta1.BundleKindHelm
	if use, _ := strconv.ParseBool(annotations[pluginscommon.AnnotationUseTemplate]); use {
		kind = pluginsv1beta1.BundleKindTemplate
	}

	maincate, cate := "other", "unknow"
	categories := strings.Split(annotations[pluginscommon.AnnotationCategory], "/")
	if len(categories) == 1 {
		cate = categories[0]
	} else if len(categories) > 1 {
		maincate, cate = categories[0], categories[1]
	}

	valsFrom := []pluginsv1beta1.ValuesFrom{}
	for _, val := range strings.Split(annotations[pluginscommon.AnnotationValuesFrom], ",") {
		if val == "" {
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
		Kind:             kind,
		Repository:       repo,
		InstallNamespace: annotations[pluginscommon.AnnotationInstallNamespace],
		Version:          cv.Version,
		Description:      cv.Description,
		MainCategory:     maincate,
		Category:         cate,
		ValuesFrom:       valsFrom,
		Required:         required,
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
