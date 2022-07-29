package models

import (
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/repository"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
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

type SourceWithSyncStatus struct {
	repository.SourceWithAddtional `json:",inline"`
	SyncStatus                     SyncStatus `json:"syncStatus"`
}

func (m *ModelsAPI) GetSource(req *restful.Request, resp *restful.Response) {
	getsourceopt := repository.GetSourceOptions{
		WithCounts: request.Query(req.Request, "count", false),
	}
	name := req.PathParameter("source")
	source, err := m.SourcesRepository.Get(req.Request.Context(), name, getsourceopt)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	// with sync status
	syncstatus, err := m.SyncService.SyncStatus(req.Request.Context(), name)
	if err != nil {
		response.Error(resp, err)
		return
	}
	response.OK(resp, &SourceWithSyncStatus{
		SourceWithAddtional: *source,
		SyncStatus:          SyncStatusFrom(syncstatus),
	})
}

func (m *ModelsAPI) ListSources(req *restful.Request, resp *restful.Response) {
	listOptions := repository.ListSourceOptions{}
	list, err := m.SourcesRepository.List(req.Request.Context(), listOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, list)
}
