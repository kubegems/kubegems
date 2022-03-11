package resourcelist

import (
	"github.com/robfig/cron/v3"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/database"
)

func NewResourceCache(db *database.Database, agents *agents.ClientSet) *ResourceCache {
	return &ResourceCache{
		DB:     db,
		Agents: agents,
	}
}

func (c *ResourceCache) Start() {
	cron := cron.New()
	if _, err := cron.AddFunc("@weekly", func() {
		if err := c.WorkloadSync(); err != nil {
			log.Error(err, "workload sync")
		}
	}); err != nil {
		log.Error(err, "add cron")
	}
	if _, err := cron.AddFunc("@daily", func() {
		if err := c.EnvironmentSync(); err != nil {
			log.Error(err, "environment sync")
		}
	}); err != nil {
		log.Error(err, "environment sync")
	}
	cron.Start()
}
