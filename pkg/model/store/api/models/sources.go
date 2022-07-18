package models

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/auth"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

func (m *ModelsAPI) ListSelectors(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ModelListOptions{
		CommonListOptions: ParseCommonListOptions(req),
		Tags:              request.Query(req.Request, "tags", []string{}),
		Framework:         req.QueryParameter("framework"),
		Source:            req.PathParameter("source"),
	}
	selectors, err := m.ModelRepository.ListSelectors(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, selectors)
}

func (m *ModelsAPI) GetSource(req *restful.Request, resp *restful.Response) {
	source, err := m.SourcesRepository.Get(req.Request.Context(), req.PathParameter("source"))
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, source)
}

type ResponseSource struct {
	repository.Source
	Count *SourceCount `json:"count,omitempty"`
}

func (m *ModelsAPI) ListSources(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionNone, func(ctx context.Context) (interface{}, error) {
		listOptions := repository.ListSourceOptions{
			CommonListOptions: ParseCommonListOptions(req),
		}
		list, err := m.SourcesRepository.List(req.Request.Context(), listOptions)
		if err != nil {
			return nil, err
		}
		total, _ := m.SourcesRepository.Count(req.Request.Context(), listOptions)
		withCount := request.Query(req.Request, "count", false)
		retlist := make([]ResponseSource, len(list))
		for i, source := range list {
			retlist[i] = ResponseSource{Source: source}
			if withCount {
				count, _ := m.countSource(req.Request.Context(), source.Name)
				retlist[i].Count = &count
			}
		}
		ret := response.Page{
			List:  retlist,
			Total: total,
			Page:  listOptions.Page,
			Size:  listOptions.Size,
		}
		return ret, nil
	})
}

type SourceCount struct {
	ModelsCount int64 `json:"modelsCount"`
	ImagesCount int64 `json:"imagesCount"`
}

func (m *ModelsAPI) countSource(ctx context.Context, source string) (SourceCount, error) {
	counts := SourceCount{}
	modelcount, err := m.ModelRepository.Count(ctx, repository.ModelListOptions{Source: source})
	if err != nil {
		return counts, err
	}
	counts.ModelsCount = modelcount
	return counts, nil
}

func (m *ModelsAPI) CreateSource(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionAdmin, func(ctx context.Context) (interface{}, error) {
		source := &repository.Source{}
		if err := req.ReadEntity(source); err != nil {
			return nil, err
		}
		if err := m.SourcesRepository.Create(ctx, source); err != nil {
			return nil, err
		}
		return source, nil
	})
}

func (m *ModelsAPI) DeleteSource(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionAdmin, func(ctx context.Context) (interface{}, error) {
		source := &repository.Source{
			Name: req.PathParameter("source"),
		}
		if err := m.SourcesRepository.Delete(ctx, source); err != nil {
			return nil, err
		}
		return source, nil
	})
}

func (m *ModelsAPI) UpdateSource(req *restful.Request, resp *restful.Response) {
	m.IfPermission(req, resp, auth.PermissionAdmin, func(ctx context.Context) (interface{}, error) {
		source := &repository.Source{}
		if err := req.ReadEntity(source); err != nil {
			return nil, err
		}
		source.Name = req.PathParameter("source")
		if err := m.SourcesRepository.Update(ctx, source); err != nil {
			return nil, err
		}
		return source, nil
	})
}

func (m *ModelsAPI) registerSourcesRoute() *route.Group {
	return route.NewGroup("/sources").Tag("sources").
		AddRoutes(
			route.GET("").To(m.ListSources).Paged().Doc("List sources").Response([]ResponseSource{}).Parameters(
				route.QueryParameter("count", "with counts in result").Optional().DataType("boolean"),
			),
			route.POST("").To(m.CreateSource).Doc("Create source").Parameters(
				route.BodyParameter("source", repository.Source{}),
			),
			route.PUT("/{source}").To(m.UpdateSource).Doc("Update source").Parameters(
				route.PathParameter("source", "source name"),
				route.BodyParameter("source", repository.Source{}),
			),
			route.GET("/{source}").To(m.GetSource).Doc("Get source").Response(repository.Source{}).Parameters(
				route.PathParameter("source", "Source name"),
			),
			route.DELETE("/{source}").To(m.DeleteSource).Doc("Delete source").Parameters(
				route.PathParameter("source", "Source name"),
			),
		)
}
