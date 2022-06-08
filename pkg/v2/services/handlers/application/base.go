package application

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-logr/logr"
	"github.com/opentracing/opentracing-go"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/v2/services/handlers"
	"kubegems.io/kubegems/pkg/v2/services/handlers/base"
)

type BaseHandler struct {
	base.BaseHandler
}

func (h *BaseHandler) GetRedis() *redis.Client {
	return h.Redis()
}

type HandlerFunc func(ctx context.Context, ref PathRef) (interface{}, error)

func (h *BaseHandler) NamedRefFunc(req *restful.Request, resp *restful.Response, body interface{}, fun HandlerFunc) {
	completes := []RefCompleteFunc{
		h.DirectRefNameFunc,
	}
	h.processfunc(req, resp, body, completes, fun)
}

func (h *BaseHandler) NoNameRefFunc(req *restful.Request, resp *restful.Response, body interface{}, fun HandlerFunc) {
	completes := []RefCompleteFunc{
		h.DirectRefNameFunc,
		func(r *restful.Request, pr *PathRef) error {
			pr.Name = ""
			return nil
		},
	}
	h.processfunc(req, resp, body, completes, fun)
}

func (h *BaseHandler) DirectRefNameFunc(req *restful.Request, ref *PathRef) error {
	ref.Tenant = req.PathParameter("tenant")
	ref.Project = req.PathParameter("project")
	ref.Env = req.PathParameter("environment")
	ref.Name = req.PathParameter("application")
	return nil
}

const ginContextKeyClusterNamespace = "CLUSTER-NAMESPACE"

type RefCompleteFunc func(*restful.Request, *PathRef) error

type ClusterNamespace struct {
	Cluster   string
	Namespace string
}

func (h *BaseHandler) processfunc(req *restful.Request, resp *restful.Response, body interface{}, completes []RefCompleteFunc, processfunc HandlerFunc) {
	ctx := req.Request.Context()

	span, ctx := opentracing.StartSpanFromContext(ctx, "start process")
	defer span.Finish()

	process := func(ctx context.Context) (interface{}, error) {
		if body != nil {
			if err := req.ReadEntity(body); err != nil {
				return nil, err
			}
		}
		ref := &PathRef{}
		for _, fun := range completes {
			if err := fun(req, ref); err != nil {
				return nil, err
			}
		}

		// 注入 logger
		ctx = logr.NewContext(ctx, log.FromContextOrDiscard(ctx).WithValues("ref", ref))
		// 注入 user
		ctx = context.WithValue(ctx, contextAuthorKey{}, &object.Signature{Name: "unknow", Email: "unknown"})
		// 注入 cluster namespace
		ctx = context.WithValue(ctx, contextClusterNamespaceKey{}, ClusterNamespace{})

		return processfunc(ctx, *ref)
	}

	data, err := process(ctx)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	// 如果未曾writer则响应 data，有的处理流程中会使用 sse 则不需要再次响应
	if data != nil {
		handlers.OK(resp, data)
	}
}
