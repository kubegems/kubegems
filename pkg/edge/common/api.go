package common

import (
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/labels"
	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/edge/tunnel"
	"kubegems.io/kubegems/pkg/utils/httputil/request"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
	"kubegems.io/kubegems/pkg/utils/route"
)

type EdgeClusterAPI struct {
	Cluster *EdgeClusterManager
	Tunnel  *tunnel.TunnelServer
}

func (a *EdgeClusterAPI) ListEdgeClusters(req *restful.Request, resp *restful.Response) {
	querylabels := request.Query(req.Request, "labels", "")
	listopt := request.GetListOptions(req.Request)
	selector, err := labels.Parse(querylabels)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	list, err := a.Cluster.ListPage(req.Request.Context(), listopt.Page, listopt.Size, selector)
	if err != nil {
		response.BadRequest(resp, err.Error())
	} else {
		response.OK(resp, list)
	}
}

type CreateClusterRequest struct {
	PrecreateOptions
}

type CreateClusterResponse struct {
	UID             string `json:"uid,omitempty"`
	ManifestAddress string `json:"manifestAddress,omitempty"`
}

func (a *EdgeClusterAPI) PreCreateEdgeCluster(req *restful.Request, resp *restful.Response) {
	options := &CreateClusterRequest{}
	if err := request.Body(req.Request, options); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}

	uid := options.UID
	if uid == "" {
		uid = uuid.NewString()
	}
	token := strings.ReplaceAll(uuid.NewString(), "-", "")

	registerurl, err := a.Cluster.PreCreate(req.Request.Context(), uid, token, options.PrecreateOptions)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	data := CreateClusterResponse{UID: uid, ManifestAddress: registerurl}
	response.OK(resp, data)
}

func (a *EdgeClusterAPI) GetEdgeCluster(req *restful.Request, resp *restful.Response) {
	uid := req.PathParameter("uid")
	cluster, err := a.Cluster.Get(req.Request.Context(), uid)
	if err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, cluster)
	}
}

func (a *EdgeClusterAPI) RemoveEdgeCluster(req *restful.Request, resp *restful.Response) {
	uid := req.PathParameter("uid")
	cluster, err := a.Cluster.Delete(req.Request.Context(), uid)
	if err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, cluster)
	}
}

func (a *EdgeClusterAPI) UpdateEdgeCluster(req *restful.Request, resp *restful.Response) {
	uid := req.PathParameter("uid")
	update := &v1beta1.EdgeCluster{}
	if err := request.Body(req.Request, update); err != nil {
		response.Error(resp, err)
		return
	}
	cluster, err := a.Cluster.Update(req.Request.Context(), uid, func(cluster *v1beta1.EdgeCluster) error {
		// update spec
		cluster.Spec = update.Spec
		// update annotations and labels
		cluster.Labels = update.Labels
		cluster.Annotations = update.Annotations
		return nil
	})
	if err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, cluster)
	}
}

func (a *EdgeClusterAPI) InstallAgentTemplate(req *restful.Request, resp *restful.Response) {
	uid, token := req.PathParameter("uid"), request.Query(req.Request, "token", "")
	rendered, err := a.Cluster.RenderInstallManifests(req.Request.Context(), uid, token)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, rendered)
}

type EdgeHubItem struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Connected bool   `json:"connected"`
}

func (a *EdgeClusterAPI) ListEdgeHubs(req *restful.Request, resp *restful.Response) {
	selector := labels.SelectorFromSet(labels.Set{
		LabelKeIsyEdgeHub: "true",
	})
	list, err := a.Cluster.List(req.Request.Context(), selector)
	if err != nil {
		response.ServerError(resp, err)
		return
	}
	ret := make([]EdgeHubItem, 0, len(list))
	for _, item := range list {
		ret = append(ret, EdgeHubItem{
			Name:      item.Name,
			Address:   item.Status.Manufacture[AnnotationKeyEdgeHubAddress],
			Connected: item.Status.Tunnel.Connected,
		})
	}
	response.OK(resp, ret)
}

func (a *EdgeClusterAPI) RegisterRoute(r *route.Group) {
	r.AddRoutes(
		route.GET("/edge-cluster/{uid}/agent-installer.yaml").To(a.InstallAgentTemplate).
			Parameters(route.QueryParameter("token", "bootstrap token")),
	).AddSubGroup(
		route.NewGroup("/edge-hubs").Tag("edge-hub").AddRoutes(
			route.GET("").To(a.ListEdgeHubs).Response([]EdgeHubItem{}),
		),
		route.NewGroup("/edge-clusters").Tag("edge-cluster").AddRoutes(
			route.GET("/").Paged().To(a.ListEdgeClusters).
				ShortDesc("list clusters").
				Parameters(route.QueryParameter("labels", "labels selector").Optional()).
				Response([]*v1beta1.EdgeCluster{}),
			route.POST("").To(a.PreCreateEdgeCluster).ShortDesc("pre create cluster").
				Parameters(route.BodyParameter("", CreateClusterRequest{})).
				Response(CreateClusterResponse{}),
			route.GET("/{uid}").To(a.GetEdgeCluster).Parameters(
				route.PathParameter("uid", "uid name"),
			),
			route.PUT("/{uid}").To(a.UpdateEdgeCluster).Parameters(
				route.PathParameter("uid", "uid name"),
			),
			route.DELETE("/{uid}").To(a.RemoveEdgeCluster).Parameters(
				route.PathParameter("uid", "uid name"),
			),
		).AddSubGroup(
			route.NewGroup("/{uid}/proxy/{path:*}").Tag("proxy").Parameters(
				route.PathParameter("uid", "uid name"),
				route.PathParameter("path", "proxy path"),
			).AddRoutes(
				route.HEAD("").To(a.Proxy),
				route.OPTIONS("").To(a.Proxy),
				route.POST("").To(a.Proxy),
				route.PATCH("").To(a.Proxy),
				route.GET("").To(a.Proxy),
				route.PUT("").To(a.Proxy),
				route.DELETE("").To(a.Proxy),
			),
		),
	)
}
