package exporter

import (
	"context"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/online"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/database"
	kprometheus "kubegems.io/pkg/utils/prometheus"
)

type AlertRuleCollector struct {
	alertrules *prometheus.Desc

	*database.Database
	*agents.ClientSet
	mutex sync.Mutex
}

func NewAlertRuleCollector(cs *agents.ClientSet, db *database.Database) Collectorfunc {
	return func(_ *log.Logger) (Collector, error) {
		return &AlertRuleCollector{
			alertrules: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "alert_rule", "status"),
				"Gems alert rule status",
				[]string{"cluster", "namespace", "name", "resource", "rule", "receiver_count"},
				nil,
			),
			ClientSet: cs,
			Database:  db,
		}, nil
	}
}

func (c *AlertRuleCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	monitoropts := new(kprometheus.MonitorOptions)
	online.LoadOptions(monitoropts, c.Database.DB())

	return c.ClientSet.ExecuteInEachCluster(context.TODO(), func(ctx context.Context, cli agents.Client) error {
		alertrules, err := cli.Extend().ListAllAlertRules(ctx, monitoropts)
		if err != nil {
			log.Error(err, "list alert rule in failed", "cluster", cli.Name())
			return nil
		}
		for _, v := range alertrules {
			ch <- prometheus.MustNewConstMetric(
				c.alertrules,
				prometheus.GaugeValue,
				utils.BoolToFloat64(v.IsOpen),
				cli.Name(), v.Namespace, v.Name, v.Resource, v.Rule, strconv.Itoa(len(v.Receivers)),
			)
		}
		return nil
	})
}
