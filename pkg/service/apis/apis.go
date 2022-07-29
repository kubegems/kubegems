package apis

import (
	"context"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store/api/modeldeployments"
	"kubegems.io/kubegems/pkg/service/aaa/auth"
	"kubegems.io/kubegems/pkg/service/apis/models"
	"kubegems.io/kubegems/pkg/service/apis/plugins"
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

	pluginsapi, err := plugins.NewPluginsAPI()
	if err != nil {
		return nil, err
	}

	modules := []apiutil.RestModule{
		modeldeployments.NewModelDeploymentAPI(deps.Agents, deps.Database),
		modelsapi,
		pluginsapi,
	}
	middlewares := []restful.FilterFunction{
		auth.NewAuthMiddleware(deps.Opts.JWT, nil).GoRestfulMiddleware, // authc
	}
	return apiutil.NewRestfulAPI("v1", middlewares, modules), nil
}
