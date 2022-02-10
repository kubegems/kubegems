package collector

import (
	"strconv"
	"strings"
	"sync"

	"github.com/kubegems/gems/pkg/agent/cluster"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/utils/exporter"
	"github.com/kubegems/gems/pkg/utils/plugins"
	"github.com/prometheus/client_golang/prometheus"
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

	gemsplugins, err := plugins.GetPlugins(c.clus)
	if err != nil {
		log.Error(err, "get plugins failed")
		return err
	}

	allPlugins := map[string]plugins.PluginDetail{}
	for k, v := range gemsplugins.Spec.CorePlugins {
		v.Type = plugins.TypeCorePlugins
		allPlugins[k] = v
	}
	for k, v := range gemsplugins.Spec.KubernetesPlugins {
		v.Type = plugins.TypeKubernetesPlugins
		allPlugins[k] = v
	}
	for pluginName, details := range allPlugins {
		var status pluginstatus
		if plugins.IsPluginHelthy(c.clus, details) {
			status = statusOK
		} else {
			status = statusUnhealthy
		}

		ch <- prometheus.MustNewConstMetric(
			c.pluginStatus,
			prometheus.GaugeValue,
			float64(status),
			details.Type, strings.ReplaceAll(pluginName, "_", "-"), details.Namespace, strconv.FormatBool(details.Enabled), details.Version,
		)
	}

	return nil
}
