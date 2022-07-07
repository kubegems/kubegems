package v2

import (
	"context"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	"kubegems.io/kubegems/pkg/service/v2/api/oam"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/httputil/apiutil"
)

type API struct{}

type Dependencies struct {
	Agents   *agents.ClientSet
	Database *database.Database
}

func InitAPI(ctx context.Context, deps Dependencies) (http.Handler, error) {
	modules := []apiutil.RestModule{
		&oam.OAM{Clientset: deps.Agents, Database: deps.Database},
	}
	return apiutil.NewRestfulAPI([]restful.FilterFunction{}, modules), nil
}
