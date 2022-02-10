package task

import (
	"github.com/kubegems/gems/pkg/handlers/application"
	"github.com/kubegems/gems/pkg/utils/argo"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/git"
	"github.com/kubegems/gems/pkg/utils/redis"
)

type ApplicationTasker struct {
	*application.ApplicationProcessor
}

func MustNewApplicationTasker(db *database.Database, gitp *git.SimpleLocalProvider, argo *argo.Client, redis *redis.Client) *ApplicationTasker {
	app := application.NewApplicationProcessor(db, gitp, argo, redis, argo.AgentsCli)
	return &ApplicationTasker{ApplicationProcessor: app}
}

func (t *ApplicationTasker) ProvideFuntions() map[string]interface{} {
	return t.ApplicationProcessor.ProvideFuntions()
}
