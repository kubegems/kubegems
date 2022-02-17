package collector

import (
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/gemsplugin"
)

type PluginCollector struct {
	pluginStatus *prometheus.Desc
	clus         cluster.Interface
	mutex        sync.Mutex
}

type pluginstatus int

const (
	statusUnhealthy pluginstatus = 0
	statusOK        pluginstatus = 1
)

func NewPluginCollectorFunc(cluster cluster.Interface) func(*log.Logger) (exporter.Collector, error) {
	return func(logger *log.Logger) (exporter.Collector, error) {
		return NewPluginCollector(logger, cluster)
	}
}

func NewPluginCollector(_ *log.Logger, clus cluster.Interface) (exporter.Collector, error) {
	c := &PluginCollector{
		pluginStatus: prometheus.NewDesc(
			prometheus.BuildFQName(exporter.GetNamespace(), "plugin", "status"),
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

	gemsplugins, err := gemsplugin.GetPlugins(c.clus)
	if err != nil {
		log.Error(err, "get plugins failed")
		return err
	}

	allPlugins := map[string]*gemsplugin.Plugin{}
	for k, v := range gemsplugins.Spec.CorePlugins {
		v.Type = gemsplugin.TypeCorePlugins
		allPlugins[k] = v
	}
	for k, v := range gemsplugins.Spec.KubernetesPlugins {
		v.Type = gemsplugin.TypeKubernetesPlugins
		allPlugins[k] = v
	}

	wg := sync.WaitGroup{}
	for pluginName, plugin := range allPlugins {
		plugin.Name = pluginName
		wg.Add(1)
		go func(p *gemsplugin.Plugin) {
			defer wg.Done()

			ch <- prometheus.MustNewConstMetric(
				c.pluginStatus,
				prometheus.GaugeValue,
				func() float64 {
					if gemsplugin.IsPluginHelthy(c.clus, p) {
						return 1
					}
					return 0
				}(),
				p.Type, p.Name, p.Namespace, strconv.FormatBool(p.Enabled), p.Version,
			)
		}(plugin)
	}
	wg.Wait()

	return nil
}
