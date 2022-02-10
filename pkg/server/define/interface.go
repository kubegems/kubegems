package define

import (
	"github.com/kubegems/gems/pkg/models"
	msgclient "github.com/kubegems/gems/pkg/msgbus/client"
	"github.com/kubegems/gems/pkg/service/aaa"
	"github.com/kubegems/gems/pkg/service/aaa/audit"
	"github.com/kubegems/gems/pkg/service/aaa/authorization"
	"github.com/kubegems/gems/pkg/service/options"
	"github.com/kubegems/gems/pkg/utils/agents"
	"github.com/kubegems/gems/pkg/utils/argo"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/redis"
	"gorm.io/gorm"
)

type (
	NotifyClient    interface{}
	ServerInterface interface {
		authorization.PermissionChecker
		audit.AuditInterface
		aaa.UserInterface
		GetOptions() *options.Options
		GetRedis() *redis.Client
		GetCacheLayer() *models.CacheLayer
		GetDB() *gorm.DB
		GetDataBase() *database.Database
		GetArgocdClient() *argo.Client
		GetAgentsClientSet() *agents.ClientSet
		GetMessageBusClient() msgclient.MessageBusInterface
	}
)
