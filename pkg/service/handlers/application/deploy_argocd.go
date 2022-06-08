package application

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/argo"
)

type SyncRequest struct {
	Resources []v1alpha1.SyncOperationResource `json:"resources,omitempty"`
}

// @Tags         Application
// @Summary      Sync同步
// @Description  Sync同步
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true   "tenaut id"
// @Param        project_id      path      int                                   true   "project id"
// @param        environment_id  path      int                                   true   "environment id"
// @Param        name            path      string                                true   "name"
// @Param        body            body      SyncRequest                           false  "指定需要同步的资源，否则全部同步"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "ok"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/sync [post]
// @Security     JWT
func (h *ApplicationHandler) Sync(c *gin.Context) {
	body := &SyncRequest{}
	h.NamedRefFunc(c, body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		h.SetAuditData(c, "同步", "应用", ref.Name)
		if err := h.ApplicationProcessor.Sync(ctx, ref, body.Resources...); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}

// @Tags         Application
// @Summary      资源树实时状态(List/Watch)
// @Description  资源树实时状态
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                             true   "tenaut id"
// @Param        project_id      path      int                                             true   "project id"
// @param        environment_id  path      int                                             true   "envid"
// @Param        name            path      string                                          true   "应用名称,应用商店名称"
// @param        watch           query     bool                                            false  "true"//  是否watch
// @Success      200             {object}  handlers.ResponseStruct{Data=ArgoResourceTree}  "summary"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/resourcetree [get]
// @Security     JWT
func (h *ApplicationHandler) ResourceTree(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		argoappname := ref.FullName()

		tree, err := h.ArgoCD.ResourceTree(ctx, argoappname)
		if err != nil {
			return nil, err
		}
		msg := h.resourceTreeListToTree(ctx, tree, h.ArgoCD, argoappname)

		iswatch, _ := strconv.ParseBool(c.Query("watch"))
		if !iswatch {
			return msg, nil
		}
		// start watching
		// list
		c.SSEvent("resourcetree", msg)
		c.Writer.Flush()
		// watch
		watchcli, err := h.ArgoCD.WatchResourceTree(ctx, argoappname)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = watchcli.CloseSend()
		}()
		// 如果出错则跳过
		c.Stream(func(_ io.Writer) bool {
			tree, err := watchcli.Recv()
			if err != nil {
				c.SSEvent("err", err.Error())
				return false
			}
			msg := h.resourceTreeListToTree(ctx, tree, h.ArgoCD, argoappname)
			c.SSEvent("resourcetree", msg)
			return true
		})
		// don't do a response
		return nil, nil
	})
}

type ArgoHistory struct {
	ID          string      `json:"id,omitempty"`
	Name        string      `json:"name,omitempty"`
	Environment string      `json:"environment,omitempty"`
	Tenant      string      `json:"tenant,omitempty"`
	Images      []string    `json:"images,omitempty"`     // 发布的镜像
	Status      string      `json:"status,omitempty"`     // 如果发布，有发布的状态，从argocd取得
	GitVersion  string      `json:"gitVersion,omitempty"` // 如果发布，有发布的 gitversion（commmit） commitid or branchname
	Publisher   string      `json:"publisher"`            // 如果发布，有发布人
	PublishAt   metav1.Time `json:"publishAt"`            // 如果发布，有发布时间 gitcommit 时间
}

// @Tags         Application
// @Summary      部署历史
// @Description  部署历史
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                                                  true  "tenaut id"
// @Param        project_id      path      int                                                                  true  "project id"
// @param        environment_id  path      int                                                                  true  "environment id"
// @Param        name            path      string                                                               true  "name"
// @Success      200             {object}  handlers.ResponseStruct{Data=handlers.PageData{Data=[]ArgoHistory}}  "history"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/argohistory [get]
// @Security     JWT
func (h *ApplicationHandler) Argohistory(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		list, err := h.listArgoHistories(ctx, ref)
		if err != nil {
			return nil, err
		}
		paged := handlers.NewPageDataFromContext(c, list, nil, nil)
		return paged, nil
	})
}

type ImageHistory struct {
	ID          string      `json:"id,omitempty"`
	Image       string      `json:"image,omitempty"`
	PublishAt   metav1.Time `json:"publishAt,omitempty"`
	Publisher   string      `json:"publisher,omitempty"`
	Environment string      `json:"environment,omitempty"` // 环境名称
	Type        string      `json:"type"`                  // 环境类型
}

// @Tags         Application
// @Summary      镜像历史
// @Description  镜像历史（生成镜像跟踪功能数据）
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                   true  "tenaut id"
// @Param        project_id      path      int                                   true  "project id"
// @param        environment_id  path      int                                   true  "environment id"
// @Param        name            path      string                                true  "name"
// @Success      200  {object}  handlers.ResponseStruct{Data=string}  "history"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/imagehistory [get]
// @Security     JWT
func (h *ApplicationHandler) ImageHistory(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		image := c.Query("image")
		if image == "" {
			return nil, fmt.Errorf("no image found in query")
		}
		argohistories, err := h.listArgoHistories(ctx, ref)
		if err != nil {
			return nil, err
		}

		list := []ImageHistory{}
		for _, his := range argohistories {
			if StringsContain(his.Images, image) {
				env, err := h.ApplicationProcessor.DataBase.GetEnvironmentWithCluster(ref)
				if err != nil {
					env = &EnvironmentDetails{}
				}
				list = append(list, ImageHistory{
					ID:          his.ID,
					Image:       image,
					PublishAt:   his.PublishAt,
					Publisher:   his.Publisher,
					Environment: his.Environment,
					Type:        env.EnvironmentType,
				})
			}
		}
		paged := handlers.NewPageDataFromContext(c, list, nil, nil)
		return paged, nil
	})
}

func StringsContain(list []string, i string) bool {
	for _, s := range list {
		if s == i {
			return true
		}
	}
	return false
}

func (h *ApplicationHandler) listArgoHistories(ctx context.Context, ref PathRef) ([]*ArgoHistory, error) {
	selector := labels.Set{
		LabelKeyFrom: LabelValueFromApp,
		LabelTenant:  ref.Tenant,
		LabelProject: ref.Project,
	}
	if ref.Env != "" {
		selector[LabelEnvironment] = ref.Env
	}
	if ref.Name != "" {
		selector[LabelApplication] = ref.Name
	}

	argoappList, err := h.ArgoCD.ListArgoApp(ctx, selector.AsSelector())
	if err != nil {
		return nil, err
	}

	ret := make([]*ArgoHistory, 0, len(argoappList.Items))
	for _, argo := range argoappList.Items {
		applicationName := argo.Labels[LabelApplication]
		env := argo.Labels[LabelEnvironment]
		tenant := argo.Labels[LabelTenant]
		// 添加当前版本

		cref := PathRef{Tenant: ref.Tenant, Project: ref.Project, Env: env, Name: applicationName}

		currentRev := argo.Status.Sync.Revision
		currentStatus := string(argo.Status.Health.Status)

		// 添加历史版本
		// 反序
		for i := len(argo.Status.History) - 1; i >= 0; i-- {
			history := argo.Status.History[i]
			item := &ArgoHistory{
				ID:          fmt.Sprintf("%s-%s-%d", env, applicationName, history.ID),
				Name:        applicationName,
				Environment: env,
				Tenant:      tenant,
				GitVersion:  history.Revision,
				Status:      "", // none
			}

			if history.DeployStartedAt != nil {
				item.PublishAt = *history.DeployStartedAt
			} else {
				item.PublishAt = history.DeployedAt
			}

			_ = h.completeArgoHistoryFromGit(ctx, cref, item)

			if item.GitVersion == currentRev {
				item.Status = currentStatus
			}

			ret = append(ret, item)
		}
	}
	return ret, nil
}

func (h *ApplicationHandler) completeArgoHistoryFromGit(ctx context.Context, ref PathRef, his *ArgoHistory) error {
	revmeta, err := h.Manifest.parseCommitImagesFunc(ctx, ref, his.GitVersion)
	if err != nil {
		return err
	}
	if his.Publisher == "" {
		his.Publisher = revmeta.Creator
	}
	if his.Images == nil {
		his.Images = revmeta.Images
	}
	if his.PublishAt.IsZero() {
		his.PublishAt = revmeta.CreatedAt
	}
	return nil
}

type ArgoResourceDiff struct {
	Group               string      `json:"group"`
	Kind                string      `json:"kind"`
	Namespace           string      `json:"namespace"`
	Name                string      `json:"name"`
	TargetState         interface{} `json:"targetState"`
	LiveState           interface{} `json:"liveState"`
	Diff                interface{} `json:"diff"` // Diff contains the JSON patch between target and live resource
	Hook                bool        `json:"hook"`
	NormalizedLiveState interface{} `json:"normalizedLiveState"`
	PredictedLiveState  interface{} `json:"predictedLiveState"`
}

// @Tags         Application
// @Summary      argo资源
// @Description  argo资源
// @Accept       json
// @Produce      json
// @Param        tenant_id       path  int     true  "tenaut id"
// @Param        project_id      path  int     true  "project id"
// @param        environment_id  path  int     true  "environment id"
// @Param        name            path  string  true  "appname"
// @params       name                  query string "resourcename"
// @params       group           query string "group"
// @params       version         query string "version"
// @params       kind                      query string "kind"
// @Success      200  {object}  handlers.ResponseStruct{Data=string}  "history"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/argoresource [get]
// @Security     JWT
func (h *ApplicationHandler) GetArgoResource(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		queries := &struct {
			Namespace string `form:"namespace" binding:"required"`
			Name      string `form:"name" binding:"required"`
			Group     string `form:"group"`
			Kind      string `form:"kind" binding:"required"`
			Version   string `form:"version"`
		}{}
		if err := c.ShouldBindQuery(queries); err != nil {
			return nil, err
		}

		argoname := ref.FullName()

		if queries.Group == v1alpha1.ApplicationSchemaGroupVersionKind.Group &&
			queries.Kind == v1alpha1.ApplicationSchemaGroupVersionKind.Kind {
			argoapp, err := h.ArgoCD.GetArgoApp(ctx, argoname)
			if err != nil {
				return nil, err
			}
			ret := ArgoResourceDiff{
				Group:     v1alpha1.ApplicationSchemaGroupVersionKind.Group,
				Kind:      v1alpha1.ApplicationSchemaGroupVersionKind.Kind,
				Namespace: argoapp.Namespace,
				Name:      argoapp.Name,
				LiveState: argoapp,
			}
			return ret, nil
		}

		// 根据请求，查询 managed resource，若存在 填充 diff livestate targetstatus
		diffresources, err := h.ArgoCD.DiffResources(ctx, &application.ResourcesQuery{
			ApplicationName: &argoname,
			Namespace:       queries.Namespace,
			Name:            queries.Name,
			Version:         queries.Version,
			Group:           queries.Group,
			Kind:            queries.Kind,
		})
		if err != nil {
			return nil, err
		}
		if len(diffresources) != 0 {
			convertArgoDiffToDiff(&v1alpha1.ResourceDiff{})
			ret := convertArgoDiffToDiff(diffresources[0])
			return ret, nil
		} else {
			req := argo.ResourceRequest{
				Name:         &argoname, // resourcename 和 name 是一样的
				ResourceName: queries.Name,
				Namespace:    queries.Namespace,
				Version:      queries.Version,
				Group:        queries.Group,
				Kind:         queries.Kind,
			}
			// 若非 managed resource，仅展示 live state
			manifest, err := h.ArgoCD.GetResource(ctx, req)
			if err != nil {
				return nil, err
			}
			ret := convertArgoDiffToDiff(&v1alpha1.ResourceDiff{
				Name:      queries.Name,
				Group:     queries.Group,
				Kind:      queries.Kind,
				Namespace: queries.Namespace,
				LiveState: manifest,
			})
			return ret, nil
		}
	})
}

func convertArgoDiffToDiff(in *v1alpha1.ResourceDiff) ArgoResourceDiff {
	stringToStruct := func(str string) map[string]interface{} {
		if str == "" {
			return nil
		}
		var data map[string]interface{}
		_ = json.Unmarshal([]byte(str), &data)
		return data
	}

	return ArgoResourceDiff{
		Group:               in.Group,
		Kind:                in.Kind,
		Namespace:           in.Namespace,
		Name:                in.Name,
		TargetState:         stringToStruct(in.TargetState),
		LiveState:           stringToStruct(in.LiveState),
		Diff:                stringToStruct(in.Diff),
		Hook:                in.Hook,
		NormalizedLiveState: stringToStruct(in.NormalizedLiveState),
		PredictedLiveState:  stringToStruct(in.PredictedLiveState),
	}
}

// ResourceTree 跟节点是 argoapp
type ArgoResourceTree struct {
	ArgoResourceNode
	Children  []*ArgoResourceTree `json:"children,omitempty"`
	LiveState interface{}         `json:"liveState,omitempty"`
}

type ArgoResourceNode struct {
	v1alpha1.ResourceNode
	Sync string `json:"sync,omitempty"`
}

func (h *ApplicationHandler) resourceTreeListToTree(ctx context.Context,
	apptree *v1alpha1.ApplicationTree, cli *argo.Client, argoappname string,
) ArgoResourceTree {
	getsyncstatus := func(argoapp *v1alpha1.Application, r v1alpha1.ResourceRef) string {
		for _, v := range argoapp.Status.Resources {
			if v.Group == r.Group && v.Kind == r.Kind && v.Namespace == r.Namespace && v.Name == r.Name {
				return string(v.Status)
			}
		}
		return ""
	}

	getchildren := func(argoapp *v1alpha1.Application, nodes []v1alpha1.ResourceNode) []*ArgoResourceTree {
		children := []*ArgoResourceTree{}
		for _, node := range nodes {
			// is root
			if len(node.ParentRefs) == 0 {
				// expand tree
				child := (&ArgoResourceTree{
					ArgoResourceNode: ArgoResourceNode{
						ResourceNode: node,
						Sync:         getsyncstatus(argoapp, node.ResourceRef),
					},
				}).fillTreeNodeChildren(nodes)
				children = append(children, child)
			}
		}
		// sort nodes
		sort.Slice(children, func(i, j int) bool {
			return !children[i].CreatedAt.Before(children[j].CreatedAt)
		})
		return children
	}

	argoappstate := v1alpha1.Application{}
	if got, err := cli.GetArgoApp(ctx, argoappname); err != nil {
		log.WithField("argo-application", argoappname).Warnf("get err %s", err.Error())
	} else {
		argoappstate = *got
	}

	return ArgoResourceTree{
		ArgoResourceNode: ArgoResourceNode{
			ResourceNode: v1alpha1.ResourceNode{
				ResourceRef: v1alpha1.ResourceRef{
					Group:     v1alpha1.SchemeGroupVersion.Group,
					Version:   v1alpha1.SchemeGroupVersion.Version,
					Kind:      v1alpha1.ApplicationSchemaGroupVersionKind.Kind,
					Name:      argoappname,
					Namespace: gemlabels.NamespaceWorkflow,
				},
				CreatedAt:       &argoappstate.CreationTimestamp,
				Health:          &argoappstate.Status.Health,
				ResourceVersion: argoappstate.ResourceVersion,
				Images:          argoappstate.Status.Summary.Images,
			},
			Sync: string(argoappstate.Status.Sync.Status),
		},
		LiveState: argoappstate,
		// 孤立节点也加入🌲
		Children: getchildren(&argoappstate, append(apptree.Nodes, apptree.OrphanedNodes...)),
	}
}

func (r *ArgoResourceTree) fillTreeNodeChildren(findchildrenFrom []v1alpha1.ResourceNode) *ArgoResourceTree {
	if r.Children == nil {
		r.Children = []*ArgoResourceTree{}
	}
	for _, child := range findchildrenFrom {
		for _, v := range child.ParentRefs {
			if isSameResourceRef(v, r.ResourceRef) {
				r.Children = append(
					r.Children,
					(&ArgoResourceTree{
						ArgoResourceNode: ArgoResourceNode{
							ResourceNode: child,
						},
					}).fillTreeNodeChildren(findchildrenFrom),
				)
			}
		}
	}
	return r
}

func isSameResourceRef(v, r v1alpha1.ResourceRef) bool {
	return v.Group == r.Group && v.Kind == r.Kind && v.Namespace == r.Namespace && v.Name == r.Name
}

// @Tags         Application
// @Summary      argo资源
// @Description  argo资源
// @Accept       json
// @Produce      json
// @Param        tenant_id       path  int     true  "tenaut id"
// @Param        project_id      path  int     true  "project id"
// @param        environment_id  path  int     true  "environment id"
// @Param        name            path  string  true  "appname"
// @params       name                       query string "resourcename"
// @params       group                 query string "group"
// @params       version               query string "version"
// @params       namespace       query string "namespace"
// @params       kind                      query string "kind"
// @Success      200             {object}  handlers.ResponseStruct{Data=string}  "history"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/environment/{environment_id}/applications/{name}/argoresource [delete]
// @Security     JWT
func (h *ApplicationHandler) DeleteArgoResource(c *gin.Context) {
	h.NamedRefFunc(c, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		queries := &struct {
			Namespace string `form:"namespace" binding:"required"`
			Name      string `form:"name" binding:"required"`
			Group     string `form:"group"`
			Kind      string `form:"kind" binding:"required"`
			Version   string `form:"version"`
		}{}
		if err := c.ShouldBindQuery(queries); err != nil {
			return nil, err
		}

		argoname := ref.FullName()

		if queries.Group == v1alpha1.ApplicationSchemaGroupVersionKind.Group &&
			queries.Kind == v1alpha1.ApplicationSchemaGroupVersionKind.Kind {
			// 删除argo 在本集群操作
			if err := h.ArgoCD.RemoveArgoApp(ctx, argoname); err != nil {
				return nil, err
			}
			return "ok", nil
		}

		req := argo.ResourceRequest{
			Name:         &argoname, // resourcename 和 name 是一样的
			ResourceName: queries.Name,
			Namespace:    queries.Namespace,
			Version:      queries.Version,
			Group:        queries.Group,
			Kind:         queries.Kind,
		}
		// 若非 managed resource，仅展示 live state
		if err := h.ArgoCD.RemoveResource(ctx, req); err != nil {
			return nil, err
		}
		return "ok", nil
	})
}
