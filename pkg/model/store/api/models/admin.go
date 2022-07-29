package models

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

func (m ModelsAPI) AdminListSources(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ListSourceOptions{
		WithDisabled:    true,
		WithModelCounts: true,
	}
	list, err := m.SourcesRepository.List(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, list)
}

func (m ModelsAPI) AdminCreateSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{}
	if err := req.ReadEntity(source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if err := m.SourcesRepository.Create(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

func (m ModelsAPI) AdminGetSource(req *restful.Request, resp *restful.Response) {
	name := req.PathParameter("source")
	getsourceopt := repository.GetSourceOptions{
		WithDisabled: true,
		WithCounts:   true,
	}
	if source, err := m.SourcesRepository.Get(req.Request.Context(), name, getsourceopt); err != nil {
		response.BadRequest(resp, err.Error())
	} else {
		response.OK(resp, source)
	}
}

func (m ModelsAPI) AdminDeleteSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{
		Name: req.PathParameter("source"),
	}
	if err := m.SourcesRepository.Delete(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

func (m ModelsAPI) AdminUpdateSource(req *restful.Request, resp *restful.Response) {
	source := &repository.Source{}
	if err := req.ReadEntity(source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	source.Name = req.PathParameter("source")
	if err := m.SourcesRepository.Update(req.Request.Context(), source); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

type SimpleModel struct {
	// nolint: tagliatelle
	ID        string   `json:"id,omitempty" bson:"_id,omitempty"`
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	Intro     string   `json:"intro"`
	Recomment int      `json:"recomment"` // number of recomment votes
	Disabled  bool     `json:"disabled"`
}

func (m ModelsAPI) AdminListModel(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
		License:           request.Query(req.Request, "license", ""),
		Task:              request.Query(req.Request, "task", ""),
		WithDisabled:      true,
	}

	list, err := m.ModelRepository.List(ctx, listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// ignore total count error
	total, _ := m.ModelRepository.Count(ctx, listOptions)
	response.OK(resp, response.Page{
		List:  list,
		Total: total,
		Page:  listOptions.Page,
		Size:  listOptions.Size,
	})
}

func (m ModelsAPI) AdminUpdateModel(req *restful.Request, resp *restful.Response) {
	model := &repository.Model{}
	if err := req.ReadEntity(model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	model.Source, model.Name = DecodeSourceModelName(req)
	if err := m.ModelRepository.Update(req.Request.Context(), model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}

func (m *ModelsAPI) registerAdminRoute() *route.Group {
	return route.
		NewGroup("/admin").Tag("admin").
		AddSubGroup(
			// source
			route.NewGroup("/sources").AddRoutes(
				route.GET("").To(m.AdminListSources).Doc("list sources").
					Response([]repository.SourceWithAddtional{}),
				route.GET("/{source}").To(m.AdminGetSource).Doc("get source").
					Parameters(route.PathParameter("source", "source name")).
					Response(repository.SourceWithAddtional{}),
				route.POST("/{source}").To(m.AdminCreateSource).Doc("add source").
					Parameters(
						route.PathParameter("source", "source name"),
						route.BodyParameter("source", repository.Source{}),
					),
				route.PUT("/{source}").To(m.AdminUpdateSource).Doc("update source").
					Parameters(
						route.PathParameter("source", "source name"),
						route.BodyParameter("source", repository.Source{}),
					).
					Response(repository.Source{}),
				route.DELETE("/{source}").To(m.AdminDeleteSource).Doc("delete source").
					Parameters(route.PathParameter("source", "source name")).
					Response(repository.Source{}),
			),
			// source sync
			route.NewGroup("/sources/{source}/sync").
				Parameters(route.PathParameter("source", "source name")).
				AddRoutes(
					route.GET("").To(m.SyncStatus).Doc("sync status").Response(SyncStatus{}),
					route.POST("").To(m.StartSync).Doc("start sync"),
					route.DELETE("").To(m.StopSync).Doc("stop sync"),
				),
			// source model
			route.
				NewGroup("/sources/{source}/models").
				Parameters(route.PathParameter("source", "source name")).
				AddRoutes(
					route.GET("").To(m.AdminListModel).Doc("list models").
						Parameters(
							route.QueryParameter("tags", "tags"),
							route.QueryParameter("framework", "framework"),
							route.QueryParameter("license", "license"),
							route.QueryParameter("task", "task"),
							route.QueryParameter("search", "search keyword"),
						).
						Paged().Response([]SimpleModel{}),
					route.PUT("/{model}").To(m.AdminUpdateModel).Doc("update model").Parameters(
						route.PathParameter("model", "model name"),
						route.BodyParameter("body", SimpleModel{}),
					),
				),
		)
}
