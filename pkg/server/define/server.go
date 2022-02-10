package define

import (
	"context"

	"gorm.io/gorm"
	"kubegems.io/pkg/models"
	msgclient "kubegems.io/pkg/msgbus/client"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/aaa/audit"
	"kubegems.io/pkg/service/aaa/authorization"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/redis"
)

type Server struct {
	Context      context.Context
	Options      *options.Options
	Cache        *redis.Client
	DB           *database.Database
	Git          *git.GitHandler
	CacheLayer   *models.CacheLayer
	ArgocdClient *argo.Client
	NotifyClient NotifyClient
	authorization.PermissionChecker
	audit.AuditInterface
	aaa.UserInterface
	MessageBusClient msgclient.MessageBusInterface
	AgentsClientSet  *agents.ClientSet
}

func (s *Server) GetMessageBusClient() msgclient.MessageBusInterface {
	// 使用新对象
	return msgclient.NewMessageBusClient(s.DB, s.GetOptions().Msgbus)
}

func (s *Server) GetContext() context.Context {
	return s.Context
}

func (s *Server) GetOptions() *options.Options {
	return s.Options
}

func (s *Server) GetRedis() *redis.Client {
	return s.Cache
}

func (s *Server) GetDB() *gorm.DB {
	return s.DB.DB()
}

func (s *Server) GetDataBase() *database.Database {
	return s.DB
}

func (s *Server) GetGit() *git.GitHandler {
	return s.Git
}

func (s *Server) GetArgocdClient() *argo.Client {
	return s.ArgocdClient
}

func (s *Server) GetCacheLayer() *models.CacheLayer {
	return s.CacheLayer
}

func (s *Server) GetPermChecker() *authorization.PermissionChecker {
	return &s.PermissionChecker
}

func (s *Server) SetCacheLayer(cacheLayer *models.CacheLayer) {
	s.CacheLayer = cacheLayer
}

func (s *Server) SetPermissionChecker(permChecker authorization.PermissionChecker) {
	s.PermissionChecker = permChecker
}

func (s *Server) SetAudit(auditInstance audit.AuditInterface) {
	s.AuditInterface = auditInstance
}

func (s *Server) GetAgentsClientSet() *agents.ClientSet {
	return s.AgentsClientSet
}
