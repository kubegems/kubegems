// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resourcelist

import (
	"github.com/robfig/cron/v3"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
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
