package exporter

import (
	"context"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/pkg/agentutils"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	kprometheus "kubegems.io/pkg/utils/prometheus"
)

type AlertRuleCollector struct {
	alertrules *prometheus.Desc

	*kprometheus.MonitorOptions
	*agents.ClientSet
	mutex sync.Mutex
}

func NewAlertRuleCollector(cs *agents.ClientSet, opts *kprometheus.MonitorOptions) Collectorfunc {
	return func(_ *log.Logger) (Collector, error) {
		return &AlertRuleCollector{
			alertrules: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "alert_rule", "status"),
				"Gems alert rule status",
				[]string{"cluster", "namespace", "name", "resource", "rule", "receiver_count"},
				nil,
			),
			ClientSet:      cs,
			MonitorOptions: opts,
		}, nil
	}
}

func (c *AlertRuleCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.ClientSet.ExecuteInEachCluster(context.TODO(), func(ctx context.Context, cli agents.Client) error {
		alertrules, err := agentutils.ListAlertRule(ctx, cli, c.MonitorOptions)
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
