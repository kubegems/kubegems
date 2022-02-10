package resourcelist

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/kubegems/gems/pkg/kubeclient"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/utils/database"
	promemodel "github.com/prometheus/common/model"
	"github.com/robfig/cron/v3"
)

func NewResourceCache(db *database.Database) *ResourceCache {
	return &ResourceCache{DB: db}
}

func (c *ResourceCache) Start() {
	cron := cron.New()
	if _, err := cron.AddFunc("@weekly", func() {
		if err := c.WorkloadSync(); err != nil {
			log.Error(err, "workload sync")
		}
	}); err != nil {
		log.Error(err, "add cron")
	}
	if _, err := cron.AddFunc("@daily", func() {
		if err := c.EnvironmentSync(); err != nil {
			log.Error(err, "environment sync")
		}
	}); err != nil {
		log.Error(err, "environment sync")
	}
	cron.Start()
}

type PromeResp struct {
	Message           string `json:"Message"`
	promemodel.Vector `json:"Data"`
	ErrorData         interface{} `json:"ErrorData"`
}

func getPrometheusResponseWithCluster(cluster, namespace, promql string) (PromeResp, error) {
	promql = strings.ReplaceAll(promql, "__namespace__", namespace)
	values := url.Values{}
	values.Add("query", promql)
	u := "/custom/prometheus/v1/vector?" + values.Encode()
	ret := promemodel.Vector{}
	err := kubeclient.DoRequest(http.MethodGet, cluster, u, nil, &ret)
	if err != nil {
		log.Error(err, "exec", "cluster", cluster, "promql", promql)
	}
	return PromeResp{Vector: ret}, err
}

// 只返回有环境标签的namespace
func collectNamespaces(cluster string) []string {
	ret := []string{}
	resp, err := getPrometheusResponseWithCluster(cluster, "", `gems_namespace_labels{environment !=""}`)
	if err != nil {
		log.Error(err, "get prometheus response")
		return ret
	}
	for _, sample := range resp.Vector {
		ns, ok := sample.Metric[NamespaceKey]
		if ok {
			ret = append(ret, string(ns))
		}
	}
	return ret
}
