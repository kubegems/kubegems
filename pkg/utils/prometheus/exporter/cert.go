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

	expiredAt, err := clusterinfo.GetServerCertExpiredTime(clusterinfo.APIServerURL, clusterinfo.APIServerCertCN)
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
