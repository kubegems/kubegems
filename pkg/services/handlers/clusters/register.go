package clusterhandler

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
)

var clusterTags = []string{"cluster"}

func (h *Handler) Regist(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/clusters")
	ws.Consumes(restful.MIME_JSON)

	ws.Route(handlers.ListCommonQuery(ws.GET("").
		To(h.List).
		Doc("list clusters").
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, nil)))

	ws.Route(ws.GET("/{cluster}").
		To(h.Retrieve).
		Doc("retireve clusters").
		Param(restful.PathParameter("detail", "is detail").PossibleValues([]string{"true", "false"})).
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.DELETE("/{cluster}").
		To(h.Delete).
		Doc("delete cluster").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.PUT("/{cluster}").
		To(h.Modify).
		Doc("modify clusters").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/plugins").
		To(h.ListPlugins).
		Doc("list cluster plugins").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.POST("/{cluster}/plugins/type/{type}/{plugin}/action/{action}").
		To(h.PluginSwitch).
		Doc("switch cluster plugins").
		Param(restful.PathParameter("name", "cluster name")).
		Param(restful.PathParameter("type", "plugin type").PossibleValues([]string{"core", "kubernetes"})).
		Param(restful.PathParameter("plugin", "plugin name")).
		Param(restful.PathParameter("action", "action name").PossibleValues([]string{"enable", "disable"})).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, nil))

	ws.Route(ws.GET("/{cluster}/environments").
		To(h.ListEnvironment).
		Doc("list cluster environments").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, []forms.EnvironmentCommon{}))

	ws.Route(ws.GET("/{cluster}/log-query-history").
		To(h.ListLogQueryHistory).
		Doc("list cluster log query history").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, []forms.LogQueryHistoryCommon{}))

	ws.Route(ws.GET("/{cluster}/log-query-snapshot").
		To(h.ListLogQueryHistory).
		Doc("list cluster log query snapshot").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, []forms.LogQuerySnapshotCommon{}))

	ws.Route(ws.GET("/{cluster}/quota-stastics").
		To(h.GetClusterQuotaStastic).
		Doc("get cluster quota stastics").
		Param(restful.PathParameter("cluster", "cluster name")).
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, []ClusterQuota{}))

	ws.Route(ws.GET("/all-status").
		To(h.ClusterStatus).
		Doc("get all cluster status").
		Metadata(restfulspec.KeyOpenAPITags, clusterTags).
		Returns(http.StatusOK, handlers.MessageOK, ClusterStatusMap{"cluster": true}))

	container.Add(ws)
}
