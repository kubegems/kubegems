package collector

import (
	"sync"

	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/utils/exporter"
	"github.com/prometheus/client_golang/prometheus"
)

type EnvironmentCollector struct {
	environmentInfo *prometheus.Desc

	mutex sync.Mutex
}

func NewEnvironmentCollector() func(_ *log.Logger) (exporter.Collector, error) {
	return func(_ *log.Logger) (exporter.Collector, error) {
		return &EnvironmentCollector{
			environmentInfo: prometheus.NewDesc(
				prometheus.BuildFQName(exporter.GetNamespace(), "environment", "info"),
				"Gems environment info",
				[]string{"environment_name", "namespace", "environment_type", "project_name", "tenant_name", "cluster_name"},
				nil,
			),
		}, nil
	}
}

func (c *EnvironmentCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var environments []models.Environment
	if err := dbinstance.DB().Preload("Project.Tenant").Preload("Cluster").Find(&environments).Error; err != nil {
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
