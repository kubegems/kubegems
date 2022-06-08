package exporter

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/kubegems/pkg/agent/cluster"
	pluginscommon "kubegems.io/kubegems/pkg/apis/plugins"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/gemsplugin"
)

type PluginCollector struct {
	pluginStatus *prometheus.Desc
	clus         cluster.Interface
	mutex        sync.Mutex
}

func NewPluginCollectorFunc(cluster cluster.Interface) func(*log.Logger) (Collector, error) {
	return func(logger *log.Logger) (Collector, error) {
		return NewPluginCollector(logger, cluster)
	}
}

func NewPluginCollector(_ *log.Logger, clus cluster.Interface) (Collector, error) {
	c := &PluginCollector{
		pluginStatus: prometheus.NewDesc(
			prometheus.BuildFQName(getNamespace(), "plugin", "status"),
			"Gems plugin status",
			[]string{"type", "plugin", "namespace", "enabled", "version"},
			nil,
		),
		clus: clus,
	}
	return c, nil
}

func (c *PluginCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, allPlugins, err := gemsplugin.ListPlugins(ctx, c.clus.GetClient(), gemsplugin.WithHealthy(true))
	if err != nil {
		log.Error(err, "get plugins failed")
		return err
	}
	for _, p := range allPlugins {
		ch <- prometheus.MustNewConstMetric(c.pluginStatus, prometheus.GaugeValue,
			func() float64 {
				if p.Healthy {
					return 1
				}
				return 0
			}(),
			p.Annotations[pluginscommon.AnnotationMainCategory], p.Name, p.Namespace, strconv.FormatBool(p.Enabled), p.Version,
		)
	}
	return nil
}
