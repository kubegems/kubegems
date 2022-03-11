package resourcelist

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/pkg/errors"
	promemodel "github.com/prometheus/common/model"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/agents"
)

const (
	environmentCPUUsageCore_LastDay_Max = `max_over_time(gems_namespace_cpu_usage_cores{environment!=""}[1d:1m])`
	environmentCPUUsageCore_LastDay_Min = `min_over_time(gems_namespace_cpu_usage_cores{environment!=""}[1d:1m])`
	environmentCPUUsageCore_LastDay_Avg = `avg_over_time(gems_namespace_cpu_usage_cores{environment!=""}[1d:1m])`

	environmentMemoryUsageByte_LastDay_Max = `max_over_time(gems_namespace_memory_usage_bytes{environment!=""}[1d:1m])`
	environmentMemoryUsageByte_LastDay_Min = `min_over_time(gems_namespace_memory_usage_bytes{environment!=""}[1d:1m])`
	environmentMemoryUsageByte_LastDay_Avg = `avg_over_time(gems_namespace_memory_usage_bytes{environment!=""}[1d:1m])`

	environmentPVCUsageByte_LastDay_Max = `max_over_time(gems_namespace_pvc_usage_bytes{environment!=""}[1d:1m])`
	environmentPVCUsageByte_LastDay_Min = `min_over_time(gems_namespace_pvc_usage_bytes{environment!=""}[1d:1m])`
	environmentPVCUsageByte_LastDay_Avg = `avg_over_time(gems_namespace_pvc_usage_bytes{environment!=""}[1d:1m])`

	environmentNetworkReceiveByte_LastDay  = `increase(gems_namespace_network_receive_bps{environment!=""}[1d])`
	environmentNetworkTransmitByte_LastDay = `increase(gems_namespace_network_send_bps{environment!=""}[1d])`

	EnvironmentKey = "environment"
	TenantKey      = "tenant"
	ProjectKey     = "project"
)

func (c *ResourceCache) EnvironmentSync() error {
	start := time.Now()

	if err := c.Agents.ExecuteInEachCluster(context.Background(), func(ctx context.Context, cli agents.Client) error {
		maxCPUUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentCPUUsageCore_LastDay_Max)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		maxMemoryUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentMemoryUsageByte_LastDay_Max)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		minCPUUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentCPUUsageCore_LastDay_Min)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		minMemoryUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentMemoryUsageByte_LastDay_Min)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		avgCPUUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentCPUUsageCore_LastDay_Avg)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		avgMemoryUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentMemoryUsageByte_LastDay_Avg)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}

		maxPVCUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentPVCUsageByte_LastDay_Max)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		minPVCUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentPVCUsageByte_LastDay_Min)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		avgPVCUsageResp, err := cli.Extend().PrometheusVector(ctx, environmentPVCUsageByte_LastDay_Avg)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}

		networkRecvResp, err := cli.Extend().PrometheusVector(ctx, environmentNetworkReceiveByte_LastDay)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}
		networkSendResp, err := cli.Extend().PrometheusVector(ctx, environmentNetworkTransmitByte_LastDay)
		if err != nil {
			return errors.Wrap(err, "failed to exec promql")
		}

		envMap := make(map[string]*models.EnvironmentResource)
		// 最大CPU使用量
		for _, sample := range maxCPUUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			p := &models.EnvironmentResource{
				ClusterName:     cli.Name(),
				TenantName:      string(sample.Metric[TenantKey]),
				ProjectName:     string(sample.Metric[ProjectKey]),
				EnvironmentName: string(sample.Metric[EnvironmentKey]),
				MaxCPUUsageCore: float64(sample.Value),
			}
			envMap[key] = p
		}

		// 最大内存使用量
		for _, sample := range maxMemoryUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.MaxMemoryUsageByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 最小CPU使用量
		for _, sample := range minCPUUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.MinCPUUsageCore = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 最小内存使用量
		for _, sample := range minMemoryUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.MinMemoryUsageByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 平均CPU使用量
		for _, sample := range avgCPUUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.AvgCPUUsageCore = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 平均内存使用量
		for _, sample := range avgMemoryUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.AvgMemoryUsageByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 最大pvc使用量
		for _, sample := range maxPVCUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.MaxPVCUsageByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 最小pvc使用量
		for _, sample := range minPVCUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.MinPVCUsageByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 平均pvc使用量
		for _, sample := range avgPVCUsageResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.AvgPVCUsageByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 网络流入
		for _, sample := range networkRecvResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.NetworkReceiveByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		// 网络流出
		for _, sample := range networkSendResp {
			key, valid := GetUniqueEnvironmentKey(sample)
			if !valid {
				log.Warnf("notvalid environment: %s", sample.Metric)
				continue
			}
			// 只处理存在的，下同
			if p, ok := envMap[key]; ok {
				p.NetworkSendByte = float64(sample.Value)
				envMap[key] = p
			}
		}

		for key := range envMap {
			if err := c.DB.DB().Save(envMap[key]).Error; err != nil {
				return errors.Wrap(err, "failed to save environment resources")
			}
		}
		return nil
	}); err != nil {
		return err
	}

	log.Info("finish environment resource list", "duration", time.Since(start).String())
	return nil
}

func GetUniqueEnvironmentKey(sample *promemodel.Sample) (key string, valid bool) {
	if sample.Metric[TenantKey] == "" || sample.Metric[ProjectKey] == "" || sample.Metric[EnvironmentKey] == "" {
		return
	}

	if math.IsInf(float64(sample.Value), 0) || math.IsNaN(float64(sample.Value)) {
		return
	}

	sample.Value = promemodel.SampleValue(utils.RoundTo(float64(sample.Value), 3))

	return fmt.Sprintf("%s_%s_%s", sample.Metric[TenantKey], sample.Metric[ProjectKey], sample.Metric[EnvironmentKey]), true
}
