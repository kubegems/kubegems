package define

import (
	"gorm.io/gorm"
	msgclient "kubegems.io/pkg/msgbus/client"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/aaa/audit"
	"kubegems.io/pkg/service/aaa/authorization"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/redis"
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
