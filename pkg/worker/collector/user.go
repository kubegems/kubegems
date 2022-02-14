package collector

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/exporter"
)

type UserCollector struct {
	userStatus *prometheus.Desc

	mutex sync.Mutex
}

func NewUserCollector() func(_ *log.Logger) (exporter.Collector, error) {
	return func(_ *log.Logger) (exporter.Collector, error) {
		return &UserCollector{
			userStatus: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetNamespace(), "user", "status"),
				"Gems user status",
				[]string{"user_name", "email", "system_role"},
				nil,
			),
		}, nil
	}
}

func (c *UserCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var users []models.User
	if err := dbinstance.DB().Preload("SystemRole").Find(&users).Error; err != nil {
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
