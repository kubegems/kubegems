// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/labels"
	route "kubegems.io/library/rest/api"
	"kubegems.io/library/rest/request"
	"kubegems.io/library/rest/response"

	"kubegems.io/kubegems/pkg/apis/edge/v1beta1"
	"kubegems.io/kubegems/pkg/edge/tunnel"
)

type EdgeClusterAPI struct {
	Cluster *EdgeManager
	Tunnel  *tunnel.TunnelServer
}

func (a *EdgeClusterAPI) ListEdgeClusters(resp http.ResponseWriter, req *http.Request) {
	querylabels, querymanufacture := request.Query(req, "labels", ""), request.Query(req, "manufacture", "")
	selector, err := labels.Parse(querylabels)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	manufacturesel, err := labels.Parse(querymanufacture)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	listopt := request.GetListOptions(req)
	total, list, err := a.Cluster.ClusterStore.List(req.Context(), ListOptions{
		Page:        listopt.Page,
		Size:        listopt.Size,
		Search:      listopt.Search,
		Selector:    selector,
		Manufacture: manufacturesel,
	})
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, response.Page[v1beta1.EdgeCluster]{
		Total: int64(total),
		List:  list,
		Page:  int64(listopt.Page),
		Size:  int64(listopt.Size),
	})
}

type CreateClusterRequest struct {
	PrecreateOptions
}

type CreateClusterResponse struct {
	UID             string `json:"uid,omitempty"`
	ManifestAddress string `json:"manifestAddress,omitempty"`
}

func (a *EdgeClusterAPI) PreCreateEdgeCluster(resp http.ResponseWriter, req *http.Request) {
	cluster := &v1beta1.EdgeCluster{}
	if err := request.Body(req, cluster); err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	if cluster.Name == "" {
		cluster.Name = uuid.NewString()
	}
	if cluster.Spec.Register.BootstrapToken == "" {
		cluster.Spec.Register.BootstrapToken = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	created, err := a.Cluster.PreCreate(req.Context(), cluster)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, created)
}

func (a *EdgeClusterAPI) GetEdgeCluster(resp http.ResponseWriter, req *http.Request) {
	uid := request.Path(req, "uid", "")
	cluster, err := a.Cluster.ClusterStore.Get(req.Context(), uid)
	if err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, cluster)
	}
}

func (a *EdgeClusterAPI) RemoveEdgeCluster(resp http.ResponseWriter, req *http.Request) {
	uid := request.Path(req, "uid", "")
	cluster, err := a.Cluster.ClusterStore.Delete(req.Context(), uid)
	if err != nil {
		response.Error(resp, err)
	} else {
		response.OK(resp, cluster)
	}
}

func (a *EdgeClusterAPI) UpdateEdgeCluster(resp http.ResponseWriter, req *http.Request) {
	uid := request.Path(req, "uid", "")
	update := &v1beta1.EdgeCluster{}
	if err := request.Body(req, update); err != nil {
		response.Error(resp, err)
		return
	}
	cluster, err := a.Cluster.ClusterStore.Update(req.Context(), uid, func(cluster *v1beta1.EdgeCluster) error {
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

func (a *EdgeClusterAPI) InstallAgentTemplate(resp http.ResponseWriter, req *http.Request) {
	uid, token := request.Path(req, "uid", ""), request.Query(req, "token", "")
	rendered, err := a.Cluster.RenderInstallManifests(req.Context(), uid, token)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.Raw(resp, http.StatusOK, rendered, nil)
}

type EdgeHubItem struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	Connected bool   `json:"connected"`
}

func (a *EdgeClusterAPI) ListEdgeHubs(resp http.ResponseWriter, req *http.Request) {
	_, list, err := a.Cluster.HubStore.List(req.Context(), ListOptions{})
	if err != nil {
		response.ServerError(resp, err)
		return
	}
	response.OK(resp, list)
}

func (a *EdgeClusterAPI) GetEdgeHub(resp http.ResponseWriter, req *http.Request) {
	uid := request.Path(req, "uid", "")
	edgehub, err := a.Cluster.HubStore.Get(req.Context(), uid)
	if err != nil {
		response.BadRequest(resp, err.Error())
		return
	}
	response.OK(resp, edgehub)
}

func (a *EdgeClusterAPI) RegisterRoute(r *route.Group) {
	r.AddRoutes(
		route.GET("/edge-clusters/{uid}/agent-installer.yaml").To(a.InstallAgentTemplate).
			Parameters(route.QueryParameter("token", "bootstrap token")),
	).AddSubGroup(
		route.NewGroup("/edge-hubs").Tag("edge-hub").AddRoutes(
			route.GET("").To(a.ListEdgeHubs).Doc("list edge hubs").
				Response([]v1beta1.EdgeHub{}),
			route.GET("/{uid}").To(a.GetEdgeHub).Parameters(
				route.PathParameter("uid", "uid hub name"),
			),
		),
		route.NewGroup("/edge-clusters").Tag("edge-cluster").AddRoutes(
			route.GET("").To(a.ListEdgeClusters).
				Doc("list clusters").
				Parameters(
					route.QueryParameter("labels", "labels selector").Optional(),
					route.QueryParameter("manufacture", "manufacture selector").Optional(),
				).
				Response([]v1beta1.EdgeCluster{}),
			route.POST("").To(a.PreCreateEdgeCluster).Doc("pre create cluster").
				Parameters(
					route.BodyParameter("", v1beta1.EdgeCluster{}),
				).
				Response(v1beta1.EdgeCluster{}),
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
			route.NewGroup("/{uid}/proxy/{path}*").Tag("proxy").Parameters(
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
