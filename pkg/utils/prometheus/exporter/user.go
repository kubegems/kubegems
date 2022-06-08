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
