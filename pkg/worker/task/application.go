package task

import (
	"kubegems.io/pkg/service/handlers/application"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/git"
	"kubegems.io/pkg/utils/redis"
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
