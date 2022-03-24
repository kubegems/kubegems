package apis

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/agent/cluster"
	pluginsv1beta1 "kubegems.io/pkg/apis/plugins/v1beta1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/gemsplugin"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PluginHandler struct {
	cluster cluster.Interface
}

type PluginsRet struct {
	CorePlugins       map[string][]*gemsplugin.Plugin `json:"core"`
	KubernetesPlugins map[string][]*gemsplugin.Plugin `json:"kubernetes"`
}

// @Tags Agent.Plugin
// @Summary 获取Plugin列表数据
// @Description 获取Plugin列表数据
// @Accept json
// @Produce json
// @Param cluster path string true "cluster"
// @Param simple query bool true "simple"
// @Success 200 {object} handlers.ResponseStruct{Data=PluginsRet} "Plugins"
// @Router /v1/proxy/cluster/{cluster}/custom/plugins.kubegems.io/v1beta1/installers [get]
// @Security JWT
func (h *PluginHandler) List(c *gin.Context) {
	allPlugins, err := gemsplugin.GetPlugins(h.cluster.Discovery())
	if err != nil {
		log.Error(err, "get plugins")
		NotOK(c, err)
		return
	}

	simple, _ := strconv.ParseBool(c.Query("simple"))
	if simple {
		ret, err := h.PluginSimple(c.Request.Context())
		if err != nil {
			NotOK(c, err)
			return
		}
		OK(c, ret)
	} else {
		ret := PluginsRet{
			CorePlugins:       make(map[string][]*gemsplugin.Plugin),
			KubernetesPlugins: make(map[string][]*gemsplugin.Plugin),
		}
		for pluginName, v := range allPlugins.Spec.CorePlugins {
			v.Status.IsHealthy = gemsplugin.IsPluginHelthy(h.cluster, v)
			v.Name = pluginName
			ret.CorePlugins[v.Details.Catalog] = append(ret.CorePlugins[v.Details.Catalog], v)
		}
		for pluginName, v := range allPlugins.Spec.KubernetesPlugins {
			v.Status.IsHealthy = gemsplugin.IsPluginHelthy(h.cluster, v)
			v.Name = pluginName
			ret.KubernetesPlugins[v.Details.Catalog] = append(ret.KubernetesPlugins[v.Details.Catalog], v)
		}

		for _, v := range ret.CorePlugins {
			sort.Slice(v, func(i, j int) bool {
				return v[i].Name < v[j].Name
			})
		}
		for _, v := range ret.KubernetesPlugins {
			sort.Slice(v, func(i, j int) bool {
				return v[i].Name < v[j].Name
			})
		}

		OK(c, ret)
	}
}

type PluginStatus map[string]bool

// plugin name -> display plugin name
// TODO: move after frontend updated
var PluginNameMapping = map[string]string{
	"argo-rollouts": "argo_rollouts",
}

func (h *PluginHandler) PluginSimple(ctx context.Context) (PluginStatus, error) {
	pluginList := &pluginsv1beta1.PluginList{}
	if err := h.cluster.GetClient().List(ctx, pluginList, &client.ListOptions{}); err != nil {
		return nil, err
	}
	ret := PluginStatus{}
	for _, plugin := range pluginList.Items {
		if retname, ok := PluginNameMapping[plugin.Name]; ok {
			ret[retname] = plugin.Spec.Enabled
		} else {
			ret[plugin.Name] = plugin.Spec.Enabled
		}
	}
	return ret, nil
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
// @Router /v1/proxy/cluster/{cluster}/custom/plugins.kubegems.io/v1beta1/installers/{name}/actions/enable [put]
// @Security JWT
func (h *PluginHandler) Enable(c *gin.Context) {
	if err := h.updatePlugin(c, func(plugin *gemsplugin.Plugin) {
		plugin.Enabled = true
	}); err != nil {
		log.Error(err, "update plugin", "plugin", c.Param("name"))
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
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
// @Router /v1/proxy/cluster/{cluster}/custom/plugins.kubegems.io/v1beta1/installers/{name}/actions/disable [put]
// @Security JWT
func (h *PluginHandler) Disable(c *gin.Context) {
	if err := h.updatePlugin(c, func(plugin *gemsplugin.Plugin) {
		plugin.Enabled = false
	}); err != nil {
		log.Error(err, "update plugin", "plugin", c.Param("name"))
		handlers.NotOK(c, err)
		return
	}
	handlers.OK(c, "ok")
}

func (h *PluginHandler) updatePlugin(
	c *gin.Context,
	mutatePlugin func(plugin *gemsplugin.Plugin),
) error {
	plugintype := c.Query("type")
	name := c.Param("name")
	allPlugins, err := gemsplugin.GetPlugins(h.cluster.Discovery())
	if err != nil {
		return err
	}
	var found *gemsplugin.Plugin
	switch plugintype {
	case gemsplugin.TypeCorePlugins:
		if v, ok := allPlugins.Spec.CorePlugins[name]; ok {
			found = v
		} else {
			return fmt.Errorf("no such plugin")
		}
	case gemsplugin.TypeKubernetesPlugins:
		if v, ok := allPlugins.Spec.KubernetesPlugins[name]; ok {
			found = v
		} else {
			return fmt.Errorf("no such plugin")
		}
	default:
		return fmt.Errorf("unknown plugin type")
	}

	mutatePlugin(found)
	return gemsplugin.UpdatePlugins(h.cluster.Discovery(), allPlugins)
}
