package resourcelist

import (
	"context"
	"net/url"
	"strings"

	promemodel "github.com/prometheus/common/model"
	"github.com/robfig/cron/v3"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/database"
)

func NewResourceCache(db *database.Database, agents *agents.ClientSet) *ResourceCache {
	return &ResourceCache{
		DB:     db,
		Agents: agents,
	}
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

func (c *ResourceCache) getPrometheusResponseWithCluster(cluster, namespace, promql string) (PromeResp, error) {
	promql = strings.ReplaceAll(promql, "__namespace__", namespace)
	values := url.Values{}
	values.Add("query", promql)

	ctx := context.TODO()

	ret := promemodel.Vector{}
	cli, err := c.Agents.ClientOf(ctx, cluster)
	if err != nil {
		return PromeResp{Vector: ret}, err
	}
	if err := cli.DoRequest(ctx, agents.Request{
		Path:  "/custom/prometheus/v1/vector",
		Query: values,
		Into:  agents.WrappedResponse(&ret),
	}); err != nil {
		log.Error(err, "exec", "cluster", cluster, "promql", promql)
	}

	return PromeResp{Vector: ret}, err
}

// 只返回有环境标签的namespace
func (c *ResourceCache) collectNamespaces(cluster string) []string {
	ret := []string{}
	resp, err := c.getPrometheusResponseWithCluster(cluster, "", `gems_namespace_labels{environment !=""}`)
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
