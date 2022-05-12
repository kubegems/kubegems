package exporter

import (
	"context"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/database"
	kprometheus "kubegems.io/pkg/utils/prometheus"
)

type AlertAndMetricCollector struct {
	alertrules    *prometheus.Desc
	metrictargets *prometheus.Desc

	cs               *agents.ClientSet
	database         *database.Database
	dyConfigProvider options.DynamicConfigurationProviderIface
	mutex            sync.Mutex
}

func NewAlertAndMetricCollector(cs *agents.ClientSet, db *database.Database, dyConfigProvider options.DynamicConfigurationProviderIface) Collectorfunc {
	return func(_ *log.Logger) (Collector, error) {
		return &AlertAndMetricCollector{
			alertrules: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "alert_rule", "status"),
				"Gems alert rule status",
				[]string{"cluster", "namespace", "name", "resource", "rule", "receiver_count", "tenant", "project", "environment"},
				nil,
			),
			metrictargets: prometheus.NewDesc(
				prometheus.BuildFQName(getNamespace(), "metric_targets", "status"),
				"Gems metric target status",
				[]string{"cluster", "namespace", "name", "target_type", "target_namespace", "target_name", "tenant", "project", "environment"},
				nil,
			),
			cs:               cs,
			database:         db,
			dyConfigProvider: dyConfigProvider,
		}, nil
	}
}

func (c *AlertAndMetricCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	monitoropts := new(kprometheus.MonitorOptions)
	c.dyConfigProvider.Get(context.Background(), monitoropts)

	envInfoMap, err := c.database.ClusterNS2EnvMap()
	if err != nil {
		return err
	}

	return c.cs.ExecuteInEachCluster(context.TODO(), func(ctx context.Context, cli agents.Client) error {
		alertrules, err := cli.Extend().ListMonitorAlertRules(ctx, corev1.NamespaceAll, monitoropts, false)
		if err != nil {
			log.Error(err, "list alert rules failed", "cluster", cli.Name())
			return nil
		}
		for _, v := range alertrules {
			envinfo := envInfoMap[cli.Name()+"/"+v.Namespace]
			ch <- prometheus.MustNewConstMetric(
				c.alertrules,
				prometheus.GaugeValue,
				utils.BoolToFloat64(v.IsOpen),
				cli.Name(), v.Namespace, v.Name, v.Resource, v.Rule, strconv.Itoa(len(v.Receivers)), envinfo.TenantName, envinfo.ProjectName, envinfo.EnvironmentName,
			)
		}

		metrictargets, err := cli.Extend().ListMetricTargets(ctx, v1.NamespaceAll)
		if err != nil {
			log.Error(err, "list metric targets failed", "cluster", cli.Name())
			return nil
		}
		for _, v := range metrictargets {
			envinfo := envInfoMap[cli.Name()+"/"+v.Namespace]
			ch <- prometheus.MustNewConstMetric(
				c.metrictargets,
				prometheus.GaugeValue,
				1,
				cli.Name(), v.Namespace, v.Name, v.TargetType, v.TargetNamespace, v.TargetName, envinfo.TenantName, envinfo.ProjectName, envinfo.EnvironmentName,
			)
		}
		return nil
	})
}
