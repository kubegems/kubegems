package exporter

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/database"
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
	for _, v := range users {
		ch <- prometheus.MustNewConstMetric(
			c.userStatus,
			prometheus.GaugeValue,
			utils.BoolToFloat64(v.IsActive),
			v.Username, v.Email, v.SystemRole.RoleCode,
		)
	}

	return nil
}
