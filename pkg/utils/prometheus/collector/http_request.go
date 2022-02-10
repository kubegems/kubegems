package collector

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/exporter"
)

type basicInfo struct {
	requestTotalCount   int64
	requestTotalSeconds float64
	start               time.Time
	countBuckets        map[float64]uint64
}

type RequestCollector struct {
	basicInfo

	requestCount *prometheus.Desc
	upDuration   *prometheus.Desc
	requestTime  *prometheus.Desc

	mutex sync.Mutex
}

func GetRequestCollector() *RequestCollector {
	tmp := exporter.GetInitiatedCollectors()
	return tmp["request"].(*RequestCollector)
}

func NewRequestCollector(_ *log.Logger) (exporter.Collector, error) {
	c := &RequestCollector{
		requestCount: prometheus.NewDesc(
			prometheus.BuildFQName(exporter.GetNamespace(), "http", "requests_total"),
			"Gems server request total",
			nil,
			nil,
		),
		upDuration: prometheus.NewDesc(
			prometheus.BuildFQName(exporter.GetNamespace(), "up", "duration_seconds"),
			"Gems server up duration",
			nil,
			nil,
		),
		requestTime: prometheus.NewDesc(
			prometheus.BuildFQName(exporter.GetNamespace(), "http", "request_duration_seconds"),
			"Gems server request duration seconds",
			nil,
			nil,
		),

		basicInfo: basicInfo{
			start: time.Now(),
			countBuckets: map[float64]uint64{
				0.005: 0,
				0.01:  0,
				0.025: 0,
				0.05:  0,
				0.1:   0,
				0.25:  0,
				0.5:   0,
				1:     0,
				2.5:   0,
				5:     0,
				10:    0,
			},
		},
		mutex: sync.Mutex{},
	}
	return c, nil
}

func (rc *RequestCollector) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		dur := time.Since(start).Seconds()

		rc.mutex.Lock()
		defer rc.mutex.Unlock()
		for k := range rc.countBuckets {
			if dur < k {
				rc.countBuckets[k] = rc.countBuckets[k] + 1
			}
		}
		rc.basicInfo.requestTotalCount++
		rc.basicInfo.requestTotalSeconds += dur
	}
}

// Update implements Collector
func (rc *RequestCollector) Update(ch chan<- prometheus.Metric) error {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()

	ch <- prometheus.MustNewConstMetric(rc.requestCount, prometheus.CounterValue, float64(rc.basicInfo.requestTotalCount))
	ch <- prometheus.MustNewConstMetric(rc.upDuration, prometheus.CounterValue, float64(time.Since(rc.start).Seconds()))
	ch <- prometheus.MustNewConstHistogram(rc.requestTime, uint64(rc.requestTotalCount), rc.requestTotalSeconds, copyBuckets(rc.countBuckets))
	return nil
}

// 复制一份 buckets map传入 Update，避免scrape时map同时读写panic
func copyBuckets(buckets map[float64]uint64) map[float64]uint64 {
	ret := map[float64]uint64{}
	for k, v := range buckets {
		ret[k] = v
	}
	return ret
}
