package plugins

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

type PluginsAPI struct {
	agents *agents.ClientSet
}

type PluginsStatus struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func NewPluginsAPI(cli *agents.ClientSet) (*PluginsAPI, error) {
	return &PluginsAPI{agents: cli}, nil
}

func (p *PluginsAPI) List(req *restful.Request, resp *restful.Response) {
	ret := []PluginsStatus{}
	ctx := req.Request.Context()
	cli, err := p.agents.ClientOfManager(ctx)
	if err != nil {
		response.Error(resp, err)
		return
	}

	globalval, plugins, err := gemsplugin.ListPlugins(ctx, cli)
	if err != nil {
		response.Error(resp, err)
		return
	}
	_ = globalval

	for _, plugin := range plugins {
		ret = append(ret, PluginsStatus{
			Name:    plugin.Name,
			Enabled: plugin.Enabled,
		})
	}
	// TODO: remove it later
	ret = append(ret, PluginsStatus{Name: "kubegems-models", Enabled: true})
	response.OK(resp, ret)
}

func (p *PluginsAPI) RegisterRoute(rg *route.Group) {
	rg.
		Tag("plugins").
		AddRoutes(
			route.GET("/plugins").To(p.List).Doc("List plugins").Response([]PluginsStatus{}),
		)
}
