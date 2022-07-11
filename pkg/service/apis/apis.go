package apis

import (
	"context"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/model/store"
	"kubegems.io/kubegems/pkg/service/apis/applications"
	"kubegems.io/kubegems/pkg/service/apis/models"
	"kubegems.io/kubegems/pkg/service/apis/oam"
	"kubegems.io/kubegems/pkg/service/handlers/application"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
	"kubegems.io/kubegems/pkg/utils/redis"
)

type API struct{}

type Dependencies struct {
	Agents   *agents.ClientSet
	Database *database.Database
	Mongo    *store.MongoDBOptions
	Gitp     *git.SimpleLocalProvider
	Argo     *argo.Client
	Redis    *redis.Client
}

func InitAPI(ctx context.Context, deps Dependencies) (http.Handler, error) {
	modelsapi, err := models.NewModelsAPI(ctx, deps.Mongo)
	if err != nil {
		return nil, err
	}
	modules := []apiutil.RestModule{
		&oam.OAM{Clientset: deps.Agents, Database: deps.Database},
		modelsapi,
		&applications.ApplicationsAPI{
			ApplicationProcessor: application.NewApplicationProcessor(deps.Database, deps.Gitp, deps.Argo, deps.Redis, deps.Agents),
		},
	}
	return apiutil.NewRestfulAPI("v1", []restful.FilterFunction{}, modules), nil
}
