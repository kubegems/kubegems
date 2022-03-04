package exporter

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/database"
)

type ClusterCollector struct {
	clusterUp *prometheus.Desc

	agents *agents.ClientSet
	*database.Database
	mutex sync.Mutex
}

func NewClusterCollector(agents *agents.ClientSet, db *database.Database) func(_ *log.Logger) (Collector, error) {
	return func(_ *log.Logger) (Collector, error) {
		return &ClusterCollector{
			agents: agents,
			clusterUp: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "cluster", "up"),
				"Gems cluster status",
				[]string{"cluster_name", "api_server", "version"},
				nil,
			),
			Database: db,
		}, nil
	}
}

func (c *ClusterCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var clusters []*models.Cluster
	if err := c.Database.DB().Find(&clusters).Error; err != nil {
		return err
	}

	return c.agents.ExecuteInEachCluster(context.TODO(), func(ctx context.Context, cli agents.Client) error {
		ishealth := true
		if err := cli.Extend().Healthy(ctx); err != nil {
			ishealth = false
		}

		addr := cli.APIServerAddr()
		ch <- prometheus.MustNewConstMetric(
			c.clusterUp,
			prometheus.CounterValue,
			utils.BoolToFloat64(ishealth),
			cli.Name(), (&addr).String(), cli.APIServerVersion(),
		)
		return nil
	})
}
