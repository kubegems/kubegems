package application

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"kubegems.io/pkg/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *ManifestHandler) GetResource(req *restful.Request, resp *restful.Response) {
	h.resourceFunc(req, resp, nil, func(ctx context.Context, gvkn GVKN, store GitStore) (interface{}, error) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvkn.GVK())
		obj.SetName(gvkn.Name)

		if err := store.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
			return nil, err
		}
		return obj, nil
	})
}

func (h *ManifestHandler) ListResources(req *restful.Request, resp *restful.Response) {
	h.resourceFunc(req, resp, nil, func(ctx context.Context, gvkn GVKN, store GitStore) (interface{}, error) {
		list := &unstructured.UnstructuredList{}
		gvk := gvkn.GVK()
		gvk.Kind = gvk.Kind + "List"
		list.SetGroupVersionKind(gvk)

		if err := store.List(ctx, list); err != nil {
			return nil, err
		}
		return list, nil
	})
}

func (h *ManifestHandler) CreateResource(req *restful.Request, resp *restful.Response) {
	obj := &unstructured.Unstructured{}
	h.resourceFunc(req, resp, obj, func(ctx context.Context, gvkn GVKN, store GitStore) (interface{}, error) {
		obj.SetGroupVersionKind(gvkn.GVK())
		obj.SetName(gvkn.Name)

		if err := store.Create(ctx, obj); err != nil {
			return nil, err
		}
		return obj, nil
	})
}

func (h *ManifestHandler) UpdateResource(req *restful.Request, resp *restful.Response) {
	obj := &unstructured.Unstructured{}
	h.resourceFunc(req, resp, obj, func(ctx context.Context, gvkn GVKN, store GitStore) (interface{}, error) {
		obj.SetGroupVersionKind(gvkn.GVK())
		obj.SetName(gvkn.Name)

		if err := store.Update(ctx, obj); err != nil {
			return nil, err
		}
		return obj, nil
	})
}

func (h *ManifestHandler) DeleteResource(req *restful.Request, resp *restful.Response) {
	h.resourceFunc(req, resp, nil, func(ctx context.Context, gvkn GVKN, store GitStore) (interface{}, error) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvkn.GVK())
		obj.SetName(gvkn.Name)

		if err := store.Delete(ctx, obj); err != nil {
			return nil, err
		}
		return obj, nil
	})
}

// @Tags         Application
// @Summary      应用内容类型摘要
// @Description  对应用内所有资源进行列举，用于自动补全
// @Accept       json
// @Produce      json
// @Param        tenant_id       path      int                                     true   "tenaut id"
// @Param        project_id      path      int                                     true   "project id"
// @Param        application_id  path      int                                     true   "application id"
// @param        environment_id  path      int                                     true   "environment id"
// @Param        name            path      string                                  true   "name"
// @Param        kind            query     string                                  false  "若设置，则仅显示设置的类型，例如 Deployment,StatefulSet,Job,ConfigMap"
// @Success      200             {object}  handlers.ResponseStruct{Data=[]object}  "类型信息"
// @Router       /v1/tenant/{tenant_id}/project/{project_id}/manifests/{name}/metas [get]
// @Security     JWT
func (h *ManifestHandler) ListMetas(req *restful.Request, resp *restful.Response) {
	h.NamedRefFunc(req, resp, nil, func(ctx context.Context, ref PathRef) (interface{}, error) {
		kind := req.QueryParameter("kind")

		ret := []client.Object{}
		fun := func(ctx context.Context, store GitStore) error {
			objects, _ := store.ListAll(ctx)
			for _, object := range objects {
				if object.GetObjectKind().GroupVersionKind().Kind != kind {
					continue
				}
				ret = append(ret, object)
			}
			return nil
		}

		if err := h.StoreFunc(ctx, ref, fun); err != nil {
			return nil, err
		}
		return ret, nil
	})
}

type GVKN struct {
	Group   string `json:"group,omitempty" uri:"group"`
	Version string `json:"version,omitempty" uri:"version"`
	Kind    string `json:"kind,omitempty" uri:"kind"`
	Name    string `json:"name,omitempty" uri:"resourcename"`
}

func (g GVKN) GVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: g.Group, Version: g.Version, Kind: g.Kind}
}

func (h *ManifestHandler) resourceFunc(req *restful.Request, resp *restful.Response, body interface{},
	gvknfun func(ctx context.Context, gvkn GVKN, store GitStore) (interface{}, error)) {
	gvkn := GVKN{
		Group:   utils.StrOrDef(req.PathParameter("group"), "core"),
		Version: req.PathParameter("version"),
		Kind:    req.PathParameter("kind"),
		Name:    req.PathParameter("name"),
	}

	h.NamedRefFunc(req, resp, body, func(ctx context.Context, ref PathRef) (interface{}, error) {
		var ret interface{}
		err := h.ManifestProcessor.StoreUpdateFunc(ctx, ref,

			func(ctx context.Context, store GitStore) error {
				data, err := gvknfun(ctx, gvkn, store)
				if err != nil {
					return err
				}
				ret = data
				return nil
			},

			"update resources",
		)
		if err != nil {
			return nil, err
		}
		return ret, nil
	})
}
