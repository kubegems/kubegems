package apis

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/emicklei/go-restful/v3"
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/model/store/api/modeldeployments"
	"kubegems.io/kubegems/pkg/service/aaa/auth"
	"kubegems.io/kubegems/pkg/service/apis/applications"
	"kubegems.io/kubegems/pkg/service/apis/models"
	"kubegems.io/kubegems/pkg/service/handlers/application"
	"kubegems.io/kubegems/pkg/service/options"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/redis"
)

type API struct{}

type Dependencies struct {
	Opts     *options.Options
	Agents   *agents.ClientSet
	Database *database.Database
	Gitp     *git.SimpleLocalProvider
	Argo     *argo.Client
	Redis    *redis.Client
}

func InitAPI(ctx context.Context, deps Dependencies) (http.Handler, error) {
	modelsapi, err := models.NewModelsAPI(ctx, deps.Opts.Mongo)
	if err != nil {
		return nil, err
	}
	modules := []apiutil.RestModule{
		modeldeployments.NewModelDeploymentAPI(deps.Agents, deps.Database),
		modelsapi,
		applications.NewApplicationsAPI(application.NewApplicationProcessor(deps.Database, deps.Gitp, deps.Argo, deps.Redis, deps.Agents)),
	}
	middlewares := []restful.FilterFunction{
		auth.NewAuthMiddleware(deps.Opts.JWT, nil).GoRestfulMiddleware, // authc
	}
	return apiutil.NewRestfulAPI("v1", middlewares, modules), nil
}

type ModelsProxy struct {
	Proxy http.Handler
}

func NewModelsProxy(target string) (*ModelsProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	return &ModelsProxy{Proxy: httputil.NewSingleHostReverseProxy(u)}, nil
}

func (p *ModelsProxy) Handler(ctx *gin.Context) {
	p.Proxy.ServeHTTP(ctx.Writer, ctx.Request)
}
