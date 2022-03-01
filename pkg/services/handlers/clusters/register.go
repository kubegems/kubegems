package clusterhandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/services/handlers"
)

var (
	clusterTags       = []string{"cluster"}
	clusterPluginTags = []string{"cluster", "cluster-plugin"}
)

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/v2/clusters")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("").
		To(h.ListCluster).
		Doc("list clusters").
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, ClusterListResp{})))

	ws.Route(ws.GET("/{cluster}").
		To(h.RetrieveCluster).
		Doc("retireve clusters").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, ClusterInfoResp{}))

	ws.Route(ws.DELETE("/{cluster}").
		To(h.DeleteCluster).
		Doc("delete cluster").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusNoContent, handlers.MessageOK, nil))

	ws.Route(ws.POST("/{cluster}").
		To(h.CreateCluster).
		Doc("create clusters").
		Reads(models.Cluster{}).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusBadRequest, "validate failed", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, ClusterResp{}))

	ws.Route(ws.PUT("/{cluster}").
		To(h.ModifyCluster).
		Doc("modify clusters").
		Reads(models.Cluster{}).
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusBadRequest, "validate failed", handlers.Response{}).
		Returns(http.StatusOK, handlers.MessageOK, ClusterResp{}))

	ws.Route(ws.GET("/{cluster}/plugins").
		To(h.ListPlugins).
		Doc("list cluster plugins").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, map[string]interface{}{}))

	ws.Route(ws.POST("/{cluster}/plugins/{plugin}/types/{type}/action/{action}").
		To(h.PluginSwitch).
		Doc("switch cluster plugins").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.PathParameter("type", "plugin type").PossibleValues([]string{"core", "kubernetes"})).
		Param(restful.PathParameter("plugin", "plugin name")).
		Param(restful.PathParameter("action", "action name").PossibleValues([]string{"enable", "disable"})).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, "ok"))

	ws.Route(ws.GET("/{cluster}/environments").
		To(h.ListEnvironment).
		Doc("list cluster environments").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, EnvironmentListResp{}))

	ws.Route(ws.GET("/{cluster}/log-query-snapshot").
		To(h.ListLogQueryHistory).
		Doc("list cluster log query snapshot").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, LogQuerySnapshotListResp{}))

	ws.Route(ws.GET("/{cluster}/quota-stastics").
		To(h.GetClusterQuotaStastic).
		Doc("get cluster quota stastics").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, ClusterQuotaResp{}))

	ws.Route(ws.GET("/all-status").
		To(h.ClusterStatus).
		Doc("get all cluster status").
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, ClusterStatusMapResp{}))

	h.registLoki(ws)
	container.Add(ws)
}

func (h *Handler) registLoki(ws *restful.WebService) {
	ws.Route(ws.GET("/{cluster}/plugin/loki/datas/queryrange").
		To(h.QueryRange).
		Doc("get cluster loki queryrange").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.QueryParameter("start", "start time")).
		Param(restful.QueryParameter("end", "end time")).
		Param(restful.QueryParameter("step", "step")).
		Param(restful.QueryParameter("interval", "interval")).
		Param(restful.QueryParameter("query", "query")).
		Param(restful.QueryParameter("direction", "direction")).
		Param(restful.QueryParameter("limit", "limit")).
		Param(restful.QueryParameter("level", "level")).
		Param(restful.QueryParameter("filters", "filters")).
		Metadata(restfulspec.KeyOpenAPITags, clusterPluginTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/plugin/loki/datas/labels").
		To(h.Labels).
		Doc("get cluster loki labels").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.QueryParameter("start", "start time")).
		Param(restful.QueryParameter("end", "end time")).
		Param(restful.QueryParameter("label", "label")).
		Metadata(restfulspec.KeyOpenAPITags, clusterPluginTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/plugin/loki/datas/export").
		To(h.Export).
		Doc("export cluster loki log").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.QueryParameter("start", "start time")).
		Param(restful.QueryParameter("end", "end time")).
		Param(restful.QueryParameter("step", "step")).
		Param(restful.QueryParameter("interval", "interval")).
		Param(restful.QueryParameter("query", "query")).
		Param(restful.QueryParameter("direction", "direction")).
		Param(restful.QueryParameter("limit", "limit")).
		Metadata(restfulspec.KeyOpenAPITags, clusterPluginTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/plugin/loki/datas/querylanguage").
		To(h.QueryLanguage).
		Doc("get cluster loki query language").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.QueryParameter("filters", "filters")).
		Param(restful.QueryParameter("pod", "pod")).
		Metadata(restfulspec.KeyOpenAPITags, clusterPluginTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/plugin/loki/datas/series").
		To(h.Series).
		Doc("get cluster loki series values").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.QueryParameter("start", "start time")).
		Param(restful.QueryParameter("end", "end time")).
		Param(restful.QueryParameter("match", "match")).
		Param(restful.QueryParameter("label", "label")).
		Metadata(restfulspec.KeyOpenAPITags, clusterPluginTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/plugin/loki/datas/context").
		To(h.Context).
		Doc("get cluster loki context").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.QueryParameter("start", "start time")).
		Param(restful.QueryParameter("end", "end time")).
		Param(restful.QueryParameter("step", "step")).
		Param(restful.QueryParameter("interval", "interval")).
		Param(restful.QueryParameter("query", "query")).
		Param(restful.QueryParameter("direction", "direction")).
		Param(restful.QueryParameter("limit", "limit")).
		Metadata(restfulspec.KeyOpenAPITags, clusterPluginTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/plugin/loki/datas/labelvalues").
		To(h.LabelValues).
		Doc("get cluster loki context").
		Param(restful.PathParameter("cluster", "cluster name")).
		Param(restful.QueryParameter("start", "start time")).
		Param(restful.QueryParameter("end", "end time")).
		Param(restful.QueryParameter("label", "label").Required(true)).
		Metadata(restfulspec.KeyOpenAPITags, clusterPluginTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))
}
