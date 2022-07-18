package models

import (
	"encoding/base64"
	"net/url"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

func DecodeSourceModelName(req *restful.Request) (string, string) {
	source := req.PathParameter("source")
	name := req.PathParameter("model")

	// model name may contains '/' so we b64encode model name at frontend
	if decoded, err := base64.StdEncoding.DecodeString(name); err == nil {
		name = string(decoded)
	}

	if decodedname, _ := url.PathUnescape(name); decodedname != "" {
		name = decodedname
	}
	return source, name
}

type ModelResponse struct {
	repository.Model
	Rating   repository.Rating `json:"rating"`
	Versions []string          `json:"versions"` // not used
}

func (m *ModelsAPI) ListModels(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
		WithRating:        request.Query(req.Request, "withRating", true),
		License:           request.Query(req.Request, "license", ""),
		Task:              request.Query(req.Request, "task", ""),
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

func (m *ModelsAPI) GetModel(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	model, err := m.ModelRepository.Get(req.Request.Context(), source, name)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}

func (m *ModelsAPI) CreateModel(req *restful.Request, resp *restful.Response) {
	model := repository.Model{}
	if err := req.ReadEntity(&model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if err := m.ModelRepository.Create(req.Request.Context(), model); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, model)
}

func (m *ModelsAPI) UpdateModel(req *restful.Request, resp *restful.Response) {
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

func (m *ModelsAPI) DeleteModel(req *restful.Request, resp *restful.Response) {
	source, name := DecodeSourceModelName(req)
	if err := m.ModelRepository.Delete(req.Request.Context(), source, name); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, nil)
}

func (m *ModelsAPI) registerModelsRoute() *route.Group {
	return route.
		NewGroup("/models").Tag("models").
		AddRoutes(
			route.GET("/{model}").To(m.GetModel).Doc("get model").Response(repository.Model{}),
			route.POST("").To(m.CreateModel).Doc("create model").Parameters(route.BodyParameter("model", repository.Model{})),
			route.GET("").To(m.ListModels).Paged().Doc("list models").Response([]repository.Model{}).Parameters(
				route.QueryParameter("framework", "framework name").Optional(),
				route.QueryParameter("license", "license name").Optional(),
				route.QueryParameter("search", "search name").Optional(),
				route.QueryParameter("tags", "filter models contains all tags").Optional(),
				route.QueryParameter("task", "task").Optional(),
				route.QueryParameter("framework", "framework").Optional(),
				route.QueryParameter("sort",
					`sort string, eg: "-name,-creationtime", "name,-creationtime"the '-' prefix means descending,otherwise ascending"`,
				).Optional(),
			),
			route.PUT("/{model}").To(m.UpdateModel).Doc("update model").Parameters(route.BodyParameter("model", repository.Model{})),
			route.DELETE("/{model}").To(m.DeleteModel).Doc("delete model"),
		)
}
