package task

import (
	"kubegems.io/kubegems/pkg/service/handlers/application"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/redis"
)

type ApplicationTasker struct {
	*application.ApplicationProcessor
}

func MustNewApplicationTasker(db *database.Database, gitp *git.SimpleLocalProvider, argo *argo.Client, redis *redis.Client, agents *agents.ClientSet) *ApplicationTasker {
	app := application.NewApplicationProcessor(db, gitp, argo, redis, agents)
	return &ApplicationTasker{ApplicationProcessor: app}
}

func (t *ApplicationTasker) ProvideFuntions() map[string]interface{} {
	return t.ApplicationProcessor.ProvideFuntions()
}
