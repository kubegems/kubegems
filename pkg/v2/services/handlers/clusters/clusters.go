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

package clusterhandler

import (
	"context"
	"fmt"
	"sync"

	"github.com/emicklei/go-restful/v3"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/v2/models"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
	"kubegems.io/kubegems/pkg/v2/services/handlers/base"
)

type Handler struct {
	base.BaseHandler
}

func (h *Handler) ListCluster(req *restful.Request, resp *restful.Response) {
	ol := &[]models.Cluster{}
	scopes := []func(*gorm.DB) *gorm.DB{
		handlers.ScopeTable(ol),
		handlers.ScopeOrder(req, []string{"create_at"}),
		handlers.ScopeSearch(req, &models.Cluster{}, []string{"name"}),
	}
	var total int64
	if err := h.DBWithContext(req).Scopes(scopes...).Count(&total).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	scopes = append(scopes, handlers.ScopePageSize(req))
	db := h.DBWithContext(req).Scopes(scopes...).Find(ol)
	if err := db.Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.OK(resp, handlers.Page(db, total, ol))
}

func (h *Handler) RetrieveCluster(req *restful.Request, resp *restful.Response) {
	cluster := &models.ClusterSimple{}
	handlers.WhereNameEqual(req.PathParameter("cluster"))
	conds := []*handlers.Cond{
		handlers.WhereNameEqual(req.PathParameter("cluster")),
	}
	tx := h.DBWithContext(req).Scopes(
		handlers.ScopeCondition(conds, cluster),
	)
	if err := tx.First(cluster).Error; err != nil {
		handlers.NotFoundOrBadRequest(resp, err)
		return
	}
	handlers.OK(resp, cluster)
}

func (h *Handler) DeleteCluster(req *restful.Request, resp *restful.Response) {
	cluster, err := h.getCluster(req.Request.Context(), req.PathParameter("cluster"))
	if err != nil {
		handlers.NoContent(resp, err)
		return
	}
	if err := h.DBWithContext(req).Delete(cluster).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.NoContent(resp, nil)
}

func (h *Handler) CreateCluster(req *restful.Request, resp *restful.Response) {
	// todo check cluster
	cluster := &models.Cluster{}
	if err := handlers.BindData(req, cluster); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	if err := h.DBWithContext(req).Create(cluster).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	handlers.Created(resp, cluster)
}

func (h *Handler) ModifyCluster(req *restful.Request, resp *restful.Response) {
	newCluster := &models.Cluster{}
	if err := handlers.BindData(req, newCluster); err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	if newCluster.Name != req.PathParameter("cluster") {
		handlers.BadRequest(resp, fmt.Errorf("cluster name is invalid"))
	}
	if err := h.DBWithContext(req).Scopes(
		handlers.ScopeCondition([]*handlers.Cond{handlers.WhereNameEqual(req.PathParameter("cluster"))}, newCluster),
	).Updates(newCluster).Error; err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// todo: check cluster data
	handlers.OK(resp, newCluster)
}

type ClusterStatusMap map[string]bool

func (h *Handler) ClusterStatus(req *restful.Request, resp *restful.Response) {
	ctx := req.Request.Context()
	clusters := []models.Cluster{}
	h.DBWithContext(req).Find(&clusters)
	eg := &errgroup.Group{}
	mu := sync.Mutex{}
	ret := ClusterStatusMap{}
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

func (h *Handler) getCluster(ctx context.Context, name string) (*models.Cluster, error) {
	cluster := &models.Cluster{}
	conds := []*handlers.Cond{handlers.WhereNameEqual(name)}
	err := h.DB().WithContext(ctx).Scopes(
		handlers.ScopeCondition(conds, cluster),
	).First(cluster).Error
	return cluster, err
}
