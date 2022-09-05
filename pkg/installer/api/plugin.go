package api

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"golang.org/x/exp/slices"
	"golang.org/x/mod/semver"
	"helm.sh/helm/v3/pkg/repo"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	pluginsv1beta1 "kubegems.io/kubegems/pkg/apis/plugins/v1beta1"
	"kubegems.io/kubegems/pkg/installer/bundle"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/kube"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type PluginStatus struct {
	Name               string   `json:"name"`
	Namespace          string   `json:"namespace"`
	Description        string   `json:"description"`
	InstalledVersion   string   `json:"installedVersion"`
	UpgradeableVersion string   `json:"upgradeableVersion"`
	AvailableVersions  []string `json:"availableVersions"`
	Required           bool     `json:"required"`
	Healthy            bool     `json:"healthy"`
	Message            string   `json:"message"`
	maincate           string
	cate               string
}

func (o *PluginsAPI) ListPlugins(req *restful.Request, resp *restful.Response) {
	plugins, err := o.manager.List(req.Request.Context())
	if err != nil {
		response.Error(resp, err)
		return
	}
	pluginstatus := []PluginStatus{}

	for _, plugin := range plugins {
		ps := PluginStatus{
			Name:        plugin.Name,
			Namespace:   plugin.Namespace,
			Description: plugin.Description,
			Required:    plugin.Required,
			maincate:    plugin.MainCategory,
			cate:        plugin.Category,
		}
		if installed := plugin.Installed; installed != nil {
			ps.InstalledVersion = installed.Version
			ps.Healthy = installed.Healthy
		}
		if upgradble := plugin.Upgradeable; upgradble != nil {
			ps.UpgradeableVersion = upgradble.Version
		}

		availableVersion := []string{}
		for _, item := range plugin.Available {
			availableVersion = append(availableVersion, item.Version)
		}
		ps.AvailableVersions = availableVersion

		pluginstatus = append(pluginstatus, ps)
	}

	mainCategoryFunc := func(t PluginStatus) string {
		return t.maincate
	}
	categoryfunc := func(t PluginStatus) string {
		return t.cate
	}

	categoryPlugins := map[string]map[string][]PluginStatus{}
	for maincategory, list := range withCategory(pluginstatus, mainCategoryFunc) {
		categorized := withCategory(list, categoryfunc)
		// sort
		for _, list := range categorized {
			sort.Slice(list, func(i, j int) bool {
				return list[i].Name < list[j].Name
			})
		}
		categoryPlugins[maincategory] = categorized
	}
	response.OK(resp, categoryPlugins)
}

func withCategory[T any](list []T, getCate func(T) string) map[string][]T {
	ret := map[string][]T{}
	for _, v := range list {
		cate := getCate(v)
		if cate == "" {
			cate = "others"
		}
		if _, ok := ret[cate]; !ok {
			ret[cate] = []T{}
		}
		ret[cate] = append(ret[cate], v)
	}
	return ret
}

func (o *PluginsAPI) GetPlugin(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	version := req.QueryParameter("version")

	pv, err := o.manager.Get(req.Request.Context(), name, version, true)
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, pv)
}

func (o *PluginsAPI) EnablePlugin(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	version := req.QueryParameter("version")

	pv := &PluginVersion{}
	if err := request.Body(req.Request, pv); err != nil {
		response.Error(resp, err)
		return
	}
	if err := o.manager.InstallOrUpdate(req.Request.Context(), name, version, pv.Values); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, pv)
}

func (o *PluginsAPI) RemovePlugin(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("name")
	if err := o.manager.UnInstall(req.Request.Context(), name); err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, "ok")
}

func NewPluginManager(namespace string, cachedir string) (*pluginManager, error) {
	cfg, err := kube.AutoClientConfig()
	if err != nil {
		return nil, err
	}
	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	return &pluginManager{namespace: namespace, cachedir: cachedir, cli: cli}, nil
}

type pluginManager struct {
	namespace string
	cachedir  string
	cli       client.Client
}

type Plugin struct {
	Name         string          `json:"name"`
	Namespace    string          `json:"namespace"`
	MainCategory string          `json:"mainCategory"`
	Category     string          `json:"category"`
	Upgradeable  *PluginVersion  `json:"upgradeable"`
	Required     bool            `json:"required"`
	Installed    *PluginVersion  `json:"installed"`
	Available    []PluginVersion `json:"available"`
	Description  string          `json:"description"`
}

type PluginVersion struct {
	Name         string                    `json:"name,omitempty"`
	Namespace    string                    `json:"namespace,omitempty"`
	Kind         pluginsv1beta1.BundleKind `json:"kind,omitempty"`
	Description  string                    `json:"description,omitempty"`
	MainCategory string                    `json:"mainCategory,omitempty"`
	Category     string                    `json:"category,omitempty"`
	// Annotations map[string]string         `json:"annotations,omitempty"`
	Repository string                `json:"repository,omitempty"`
	URL        string                `json:"url,omitempty"`
	Version    string                `json:"version,omitempty"`
	Healthy    bool                  `json:"healthy,omitempty"`
	Required   bool                  `json:"required,omitempty"`
	Message    string                `json:"message,omitempty"`
	Values     pluginsv1beta1.Values `json:"values,omitempty"`
	Schema     string                `json:"schema,omitempty"`
}

func (m *pluginManager) List(ctx context.Context) ([]Plugin, error) {
	plugins, err := m.listPlugins(ctx)
	if err != nil {
		return nil, err
	}
	ret := []Plugin{}
	for _, plugin := range plugins {
		ret = append(ret, plugin)
	}
	slices.SortFunc(ret, func(a, b Plugin) bool {
		return strings.Compare(a.Name, b.Name) > -1
	})
	return ret, nil
}

func (m *pluginManager) Get(ctx context.Context, name, version string, withschema bool) (*PluginVersion, error) {
	plugins, err := m.listPlugins(ctx)
	if err != nil {
		return nil, err
	}
	plugin, ok := plugins[name]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	allversions := plugin.Available
	if plugin.Installed != nil {
		// if no version speci always return installed version
		allversions = append([]PluginVersion{*plugin.Installed}, allversions...)
	}

	for _, pv := range allversions {
		// if  no version we use the first version
		if version == "" || pv.Version == version {
			if withschema {
				// find schema
				intodir := filepath.Join(m.cachedir, fmt.Sprintf("%s-%s", name, version))
				_, chart, err := bundle.DownloadHelmChart(ctx, pv.Repository, pv.Name, pv.Version, intodir)
				if err != nil {
					// return nil, err
				} else {
					pv.Schema = string(chart.Schema)
				}
			}
			return &pv, nil
		}
	}
	return nil, fmt.Errorf("plugin %s version %s not found", name, version)
}

func (m *pluginManager) InstallOrUpdate(
	ctx context.Context, name string, version string, values pluginsv1beta1.Values,
) error {
	// get from remote repos
	pv, err := m.Get(ctx, name, version, false)
	if err != nil {
		return err
	}
	pv.Values = values
	apiplugin := PluginVersionTo(*pv)
	apiplugin.Namespace = m.namespace

	plugincr := &pluginsv1beta1.Plugin{ObjectMeta: v1.ObjectMeta{Name: name, Namespace: m.namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, m.cli, plugincr, func() error {
		plugincr.Annotations = apiplugin.Annotations
		plugincr.Spec = apiplugin.Spec
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *pluginManager) UnInstall(ctx context.Context, name string) error {
	plugincr := &pluginsv1beta1.Plugin{ObjectMeta: v1.ObjectMeta{Name: name, Namespace: m.namespace}}
	return m.cli.Delete(ctx, plugincr)
}

func (m *pluginManager) listPlugins(ctx context.Context) (map[string]Plugin, error) {
	// list local
	installversions, err := m.listInstalled(ctx, m.namespace)
	if err != nil {
		return nil, err
	}
	// list remotes
	avaliableversions, err := m.listRemotes(ctx)
	if err != nil {
		return nil, err
	}

	plugins := map[string]Plugin{}
	for name, available := range avaliableversions {
		p := Plugin{
			Name:      name,
			Available: available,
		}
		latestversion := available[0]
		if installed, ok := installversions[name]; ok {
			p.Installed = &installed
			// check upgrade
			if semver.Compare(installed.Version, latestversion.Version) < -1 {
				p.Upgradeable = &latestversion
			}
			delete(installversions, name)
		}
		p.Description = latestversion.Description
		p.MainCategory = latestversion.MainCategory
		p.Category = latestversion.Category
		p.Required = latestversion.Required
		p.Namespace = latestversion.Namespace
		plugins[name] = p
	}

	// installed but not in remotes
	for name, val := range installversions {
		installed := val
		plugins[name] = Plugin{
			Name:         name,
			Installed:    &installed,
			MainCategory: installed.MainCategory,
			Category:     installed.Category,
			Description:  installed.Description,
		}
	}
	return plugins, nil
}

func (m *pluginManager) listInstalled(ctx context.Context, namespace string) (map[string]PluginVersion, error) {
	pluginList := &pluginsv1beta1.PluginList{}
	if err := m.cli.List(ctx, pluginList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	ret := map[string]PluginVersion{}
	for _, plugin := range pluginList.Items {
		ret[plugin.Name] = PluginVersionFrom(&plugin)
	}
	return ret, nil
}

func (m *pluginManager) listRemotes(ctx context.Context) (map[string][]PluginVersion, error) {
	repos, err := m.ListRepo(ctx)
	if err != nil {
		return nil, err
	}
	ret := map[string][]PluginVersion{}
	for _, repo := range repos {
		for name, chartversions := range repo.Index.Entries {
			if len(chartversions) == 0 {
				continue
			}
			pvs := make([]PluginVersion, 0, len(chartversions))
			for _, cv := range chartversions {
				pvs = append(pvs, PluginVersionFromRepoChartVersion(repo.Address, cv))
			}
			if pluginversions, ok := ret[name]; ok {
				ret[name] = append(pluginversions, pvs...)
			} else {
				ret[name] = pvs
			}
		}
	}
	for name := range ret {
		slices.SortFunc(ret[name], func(a, b PluginVersion) bool {
			return semver.Compare(a.Version, b.Version) > -1
		})
	}
	return ret, nil
}

func ParseChartURL(repo string, cv *repo.ChartVersion) string {
	if len(cv.URLs) == 0 {
		return repo
	}
	u, err := url.Parse(cv.URLs[0])
	if err != nil {
		return repo
	}
	if u.Host == "" {
		return repo + "/" + u.String()
	}
	return u.String()
}

func PluginVersionFrom(plugin *pluginsv1beta1.Plugin) PluginVersion {
	annotations := plugin.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	version := plugin.Spec.Version
	if version == "" {
		version = plugin.Status.Version
	}
	if version == "" {
		version = "unknow"
	}
	return PluginVersion{
		Name:         plugin.Name,
		Namespace:    plugin.Spec.InstallNamespace,
		Kind:         plugin.Spec.Kind,
		URL:          plugin.Spec.URL,
		Version:      version,
		Message:      plugin.Status.Message,
		Values:       plugin.Spec.Values,
		Description:  annotations[pluginscommon.AnnotationDescription],
		MainCategory: annotations[pluginscommon.AnnotationMainCategory],
		Category:     annotations[pluginscommon.AnnotationCategory],
		Schema:       annotations[pluginscommon.AnnotationSchema],
		Healthy:      true,
	}
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
	return PluginVersion{
		Name:         cv.Name,
		Kind:         kind,
		URL:          ParseChartURL(repo, cv),
		Repository:   repo,
		Version:      cv.Version,
		Description:  cv.Description,
		Namespace:    annotations[pluginscommon.AnnotationInstallNamespace],
		MainCategory: annotations[pluginscommon.AnnotationMainCategory],
		Category:     annotations[pluginscommon.AnnotationCategory],
		Schema:       annotations[pluginscommon.AnnotationSchema],
		Required:     required,
		Healthy:      true,
	}
}

func PluginVersionTo(pluginversion PluginVersion) *pluginsv1beta1.Plugin {
	annotations := map[string]string{
		pluginscommon.AnnotationDescription:  pluginversion.Description,
		pluginscommon.AnnotationCategory:     pluginversion.Category,
		pluginscommon.AnnotationMainCategory: pluginversion.MainCategory,
		pluginscommon.AnnotationSchema:       pluginversion.Schema,
	}

	return &pluginsv1beta1.Plugin{
		ObjectMeta: v1.ObjectMeta{
			Name:        pluginversion.Name,
			Annotations: annotations,
		},
		Spec: pluginsv1beta1.PluginSpec{
			Kind:             pluginversion.Kind,
			URL:              pluginversion.URL,
			InstallNamespace: pluginversion.Namespace,
			Version:          pluginversion.Version,
			Values:           pluginversion.Values,
			ValuesFrom:       []pluginsv1beta1.ValuesFrom{}, // todo
		},
	}
}
