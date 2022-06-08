package exporter

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
)

type EnvironmentCollector struct {
	environmentInfo *prometheus.Desc

	*database.Database
	mutex sync.Mutex
}

func NewEnvironmentCollector(db *database.Database) func(_ *log.Logger) (Collector, error) {
	return func(_ *log.Logger) (Collector, error) {
		return &EnvironmentCollector{
			environmentInfo: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "environment", "info"),
				"Gems environment info",
				[]string{"environment", "namespace", "environment_type", "project", "tenant", "cluster"},
				nil,
			),
			Database: db,
		}, nil
	}
}

func (c *EnvironmentCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var environments []models.Environment
	if err := c.Database.DB().Preload("Project.Tenant").Preload("Cluster").Find(&environments).Error; err != nil {
		return err
	}

	for _, env := range environments {
		ch <- prometheus.MustNewConstMetric(
			c.environmentInfo,
			prometheus.GaugeValue,
			1,
			env.EnvironmentName, env.Namespace, env.MetaType, env.Project.ProjectName, env.Project.Tenant.TenantName, env.Cluster.ClusterName,
		)
	}
	return nil
}
