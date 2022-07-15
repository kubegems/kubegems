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

package exporter

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
)

type UserCollector struct {
	userStatus *prometheus.Desc

	*database.Database
	mutex sync.Mutex
}

func NewUserCollector(db *database.Database) Collectorfunc {
	return func(_ *log.Logger) (Collector, error) {
		return &UserCollector{
			userStatus: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "user", "status"),
				"Gems user status",
				[]string{"user_name", "email", "system_role"},
				nil,
			),
			Database: db,
		}, nil
	}
}

func (c *UserCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var users []models.User
	if err := c.Database.DB().Preload("SystemRole").Find(&users).Error; err != nil {
		return err
	}
	for i := range users {
		ch <- prometheus.MustNewConstMetric(
			c.userStatus,
			prometheus.GaugeValue,
			func() float64 {
				if users[i].IsActive != nil && *users[i].IsActive {
					return 1
				}
				return 0
			}(),
			users[i].Username, users[i].Email, users[i].SystemRole.RoleCode,
		)
	}

	return nil
}
