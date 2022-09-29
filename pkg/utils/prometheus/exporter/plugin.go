// Copyright 2022 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package exporter

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/kubegems/pkg/agent/cluster"
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

	pm := gemsplugin.PluginManager{Client: c.clus.GetClient()}
	allPlugins, err := pm.ListInstalled(ctx, true)
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
			p.MainCategory, p.Name, p.Namespace, strconv.FormatBool(p.Enabled), p.Version,
		)
	}
	return nil
}
