package clusterhandler

import (
	"context"
	"fmt"
	"sync"

	"github.com/emicklei/go-restful/v3"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/model/client"
	"kubegems.io/pkg/model/forms"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/services/handlers/base"
)

type Handler struct {
	base.BaseHandler
}

func (h *Handler) List(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.ClusterCommonList{}
	if err := h.Model().List(ctx, ol.Object(), handlers.CommonOptions(req)...); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(ol.Object(), ol.Data()))
}

func (h *Handler) Retrieve(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	data, err := h.getCluster(ctx, req.PathParameter("cluster"), req.QueryParameter("detail") == "true")
	if err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	handlers.OK(resp, data.DataPtr())
}

func (h *Handler) Delete(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	cluster, err := h.getClusterDetail(ctx, req.PathParameter("cluster"))
	if err != nil {
		handlers.NoContent(resp, err)
		return
	}
	if err := h.Model().Delete(ctx, cluster.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) Create(req *restful.Request, resp *restful.Response) {
	// todo check cluster
	ctx := req.Request.Context()
	cluster := &forms.ClusterDetail{}
	if err := handlers.BindData(req, cluster); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	_, err := h.getCluster(ctx, cluster.Name, false)
	if !handlers.IsNotFound(err) {
		handlers.BadRequest(resp, fmt.Errorf("exist"))
		return
	}
	if err := h.Model().Create(ctx, cluster.Object()); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, cluster.Data())
}

func (h *Handler) Modify(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	newCluster := &forms.ClusterDetail{}
	if err := handlers.BindData(req, newCluster); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	if newCluster.Name != req.PathParameter("cluster") {
		handlers.BadRequest(resp, fmt.Errorf("cluster name is invalid"))
	}
	newCluster.ID = 0
	if err := h.Model().Update(ctx, newCluster.Object(), client.WhereNameEqual(newCluster.Name)); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// todo: check cluster data
	handlers.OK(resp, newCluster.Data())
}

type ClusterStatusMap map[string]bool

func (h *Handler) ClusterStatus(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	ol := forms.ClusterCommonList{}
	h.Model().List(ctx, ol.Object())
	eg := &errgroup.Group{}
	mu := sync.Mutex{}
	ret := ClusterStatusMap{}
	clusters := ol.Data()
	for _, cluster := range clusters {
		name := cluster.Name
		eg.Go(func() error {
			cli, err := h.Agents().ClientOf(ctx, name)
			if err != nil {
				log.Error(err, "failed to get agents client", "cluster", name)
				return nil
			}
			if err := cli.Extend().Healthy(ctx); err != nil {
				log.Error(err, "cluster is unhealthy", "cluster", name)
				return nil
			}
			mu.Lock()
			defer mu.Unlock()
			ret[name] = true
			return nil
		})
	}
	_ = eg.Wait()
	handlers.OK(resp, ret)
}

func (h *Handler) getCluster(ctx context.Context, name string, detail bool) (forms.FormInterface, error) {
	if detail {
		return h.getClusterDetail(ctx, name)
	} else {
		return h.getClusterCommon(ctx, name)
	}
}

func (h *Handler) getClusterDetail(ctx context.Context, name string) (*forms.ClusterDetail, error) {
	cluster := &forms.ClusterDetail{}
	return cluster, h.Model().Get(ctx, cluster.Object(), client.WhereNameEqual(name))
}

func (h *Handler) getClusterCommon(ctx context.Context, name string) (*forms.ClusterCommon, error) {
	cluster := &forms.ClusterCommon{}
	return cluster, h.Model().Get(ctx, cluster.Object(), client.WhereNameEqual(name))
}
