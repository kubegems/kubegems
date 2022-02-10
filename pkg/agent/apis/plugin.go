package apis

import (
	"errors"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/agent/cluster"
	"github.com/kubegems/gems/pkg/utils/plugins"
)

type PluginHandler struct {
	cluster cluster.Interface
}

type PluginsRet struct {
	CorePlugins       map[string]PluginCollect `json:"core"`
	KubernetesPlugins map[string]PluginCollect `json:"kubernetes"`
}

type PluginCollect []plugins.PluginDetail

func (a PluginCollect) Len() int           { return len(a) }
func (a PluginCollect) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a PluginCollect) Less(i, j int) bool { return a[i].Name < a[j].Name }

// @Tags Agent.Plugin
// @Summary 获取Plugin列表数据
// @Description 获取Plugin列表数据
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param simple query bool true "simple"
// @Success 200 {object} handlers.ResponseStruct{Data=PluginsRet} "Plugins"
// @Router /v1/proxy/cluster/{cluster}/custom/plugins.gems.cloudminds.com/v1alpha1/plugins [get]
// @Security JWT
func (h *PluginHandler) List(c *gin.Context) {
	gemsplugins, err := plugins.GetPlugins(h.cluster)
	if err != nil {
		NotOK(c, err)
		return
	}

	simple, _ := strconv.ParseBool(c.Query("simple"))
	if simple {
		ret := make(map[string]bool)
		for name, v := range gemsplugins.Spec.CorePlugins {
			ret[name] = v.Enabled
		}
		for name, v := range gemsplugins.Spec.KubernetesPlugins {
			ret[name] = v.Enabled
		}
		OK(c, ret)
	} else {
		ret := PluginsRet{
			CorePlugins:       make(map[string]PluginCollect),
			KubernetesPlugins: make(map[string]PluginCollect),
		}
		for name, v := range gemsplugins.Spec.CorePlugins {
			v.Status.IsHealthy = plugins.IsPluginHelthy(h.cluster, v)
			v.Name = name
			if plugindetails, ok := ret.CorePlugins[v.Catalog]; ok {
				plugindetails = append(plugindetails, v)
				ret.CorePlugins[v.Catalog] = plugindetails
			} else {
				ret.CorePlugins[v.Catalog] = PluginCollect{v}
			}
		}

		for name, v := range gemsplugins.Spec.KubernetesPlugins {
			v.Status.IsHealthy = plugins.IsPluginHelthy(h.cluster, v)
			v.Name = name
			if plugindetails, ok := ret.KubernetesPlugins[v.Catalog]; ok {
				plugindetails = append(plugindetails, v)
				ret.KubernetesPlugins[v.Catalog] = plugindetails
			} else {
				ret.KubernetesPlugins[v.Catalog] = PluginCollect{v}
			}
		}

		for k, v := range ret.CorePlugins {
			sort.Sort(v)
			ret.CorePlugins[k] = v
		}

		for k, v := range ret.KubernetesPlugins {
			sort.Sort(v)
			ret.KubernetesPlugins[k] = v
		}

		OK(c, ret)
	}
}

// @Tags Agent.Plugin
// @Summary 启用插件
// @Description 启用插件
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param name path string true "name"
// @Param type query string true "type"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "Plugins"
// @Router /v1/proxy/cluster/{cluster}/custom/plugins.gems.cloudminds.com/v1alpha1/plugins/{name}/actions/enable [post]
// @Security JWT
func (h *PluginHandler) Enable(c *gin.Context) {
	plugintype := c.Query("type")
	name := c.Param("name")

	gemsplugins, err := plugins.GetPlugins(h.cluster)
	if err != nil {
		NotOK(c, err)
		return
	}

	switch plugintype {
	case plugins.TypeCorePlugins:
		if v, ok := gemsplugins.Spec.CorePlugins[name]; ok {
			v.Enabled = true
			gemsplugins.Spec.CorePlugins[name] = v
		} else {
			NotOK(c, errors.New("no such plugin"))
			return
		}
	case plugins.TypeKubernetesPlugins:
		if v, ok := gemsplugins.Spec.KubernetesPlugins[name]; ok {
			v.Enabled = true
			gemsplugins.Spec.KubernetesPlugins[name] = v
		} else {
			NotOK(c, errors.New("no such plugin"))
			return
		}
	default:
		NotOK(c, errors.New("unknown plugin type"))
		return
	}

	if err := plugins.UpdatePlugins(h.cluster, gemsplugins); err != nil {
		NotOK(c, err)
	}

	OK(c, "")
}

// @Tags Agent.Plugin
// @Summary 禁用插件
// @Description 禁用插件
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param name path string true "name"
// @Param type query string true "type"
// @Success 200 {object} handlers.ResponseStruct{Data=string} "Plugins"
// @Router /v1/proxy/cluster/{cluster}/custom/plugins.gems.cloudminds.com/v1alpha1/plugins/{name}/actions/disable [post]
// @Security JWT
func (h *PluginHandler) Disable(c *gin.Context) {
	plugintype := c.Query("type")
	name := c.Param("name")

	gemsplugins, err := plugins.GetPlugins(h.cluster)
	if err != nil {
		NotOK(c, err)
		return
	}

	switch plugintype {
	case plugins.TypeCorePlugins:
		if v, ok := gemsplugins.Spec.CorePlugins[name]; ok {
			v.Enabled = false
			gemsplugins.Spec.CorePlugins[name] = v
		} else {
			NotOK(c, errors.New("no such plugin"))
			return
		}
	case plugins.TypeKubernetesPlugins:
		if v, ok := gemsplugins.Spec.KubernetesPlugins[name]; ok {
			v.Enabled = false
			gemsplugins.Spec.KubernetesPlugins[name] = v
		} else {
			NotOK(c, errors.New("no such plugin"))
			return
		}
	default:
		NotOK(c, errors.New("unknown plugin type"))
		return
	}

	if err := plugins.UpdatePlugins(h.cluster, gemsplugins); err != nil {
		NotOK(c, err)
		return
	}

	OK(c, "")
}
