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
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/clusterinfo"
)

type CertCollector struct {
	certExpiredAt *prometheus.Desc
	mutex         sync.Mutex
}

func NewCertCollectorFunc() func(*log.Logger) (Collector, error) {
	return func(logger *log.Logger) (Collector, error) {
		return NewCertCollector(logger)
	}
}

func NewCertCollector(_ *log.Logger) (Collector, error) {
	c := &CertCollector{
		certExpiredAt: prometheus.NewDesc(
			prometheus.BuildFQName(getNamespace(), "cluster_component_cert", "expiration_remain_seconds"),
			"Gems cluster component cert expiration remain seconds",
			[]string{"component"},
			nil,
		),
	}
	return c, nil
}

func (c *CertCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	expiredAt, err := clusterinfo.GetServerCertExpiredTime(clusterinfo.APIServerURL)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(
		c.certExpiredAt,
		prometheus.GaugeValue,
		time.Until(*expiredAt).Seconds(),
		"apiserver",
	)

	return nil
}
