package define

import (
	"context"

	"github.com/kubegems/gems/pkg/aaa"
	"github.com/kubegems/gems/pkg/aaa/audit"
	"github.com/kubegems/gems/pkg/aaa/authorization"
	"github.com/kubegems/gems/pkg/models"
	msgclient "github.com/kubegems/gems/pkg/msgbus/client"
	"github.com/kubegems/gems/pkg/service/options"
	"github.com/kubegems/gems/pkg/utils/agents"
	"github.com/kubegems/gems/pkg/utils/argo"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/git"
	"github.com/kubegems/gems/pkg/utils/redis"
	"gorm.io/gorm"
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
