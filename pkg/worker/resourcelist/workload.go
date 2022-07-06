package resourcelist

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pkg/errors"
	promemodel "github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/database"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// containerCPUPercent_LastWeek 使用率超过60%
	containerCPUPercent_LastWeek = `
    quantile_over_time(0.95, gems_container_cpu_usage_percent{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}[1w:5m]) / 100 < 0.1
	or
    quantile_over_time(0.95, gems_container_cpu_usage_percent{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}[1w:5m]) / 100 > 0.6`
	containerMemoryPercent_LastWeek = `
    quantile_over_time(0.95, gems_container_memory_usage_percent{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}[1w:5m]) / 100 < 0.1
	or
    quantile_over_time(0.95, gems_container_memory_usage_percent{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}[1w:5m]) / 100 > 0.6`

	containerCPUUsageCore_LastWeek     = `quantile_over_time(0.95, gems_container_cpu_usage_cores{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}[1w:5m])`
	containerMemoryUsageBytes_LastWeek = `quantile_over_time(0.95, gems_container_memory_usage_bytes{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}[1w:5m])`

	containerCPULimitCore     = `gems_container_cpu_limit_cores{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}`
	containerMemoryLimitBytes = `gems_container_memory_limit_bytes{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"}`

	// cpu内存限制方差
	// 计算workload所有副本的平均cpu、内存变化，而不是workload的总变化，避免副本数变化带来的影响
	workloadCPULimitStdVar = `
	stdvar_over_time((sum(gems_container_cpu_limit_cores{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"})by(namespace, owner_kind, workload) 
	/ sum(gems_pod_workload{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet"})by(namespace, owner_kind, workload))[1w:5m])`
	workloadMemoryLimitStdVar = `
	stdvar_over_time((sum(gems_container_memory_limit_bytes{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet", container!~"istio-proxy|"})by(namespace, owner_kind, workload) 
	/ sum(gems_pod_workload{namespace="%[1]s", owner_kind=~"Deployment|StatefulSet|DaemonSet"})by(namespace, owner_kind, workload))[1w:5m])`

	Deployment  = "Deployment"
	StatefulSet = "StatefulSet"
	DaemonSet   = "DaemonSet"

	NamespaceKey    = "namespace"
	WorkloadTypeKey = "owner_kind"
	WorkloadNameKey = "workload"
	PodKey          = "pod"
	ContainerKey    = "container"
)

type ResourceCache struct {
	DB     *database.Database
	Agents *agents.ClientSet
}

func (c *ResourceCache) WorkloadSync() error {
	log.Info("start workload sync")
	start := time.Now()
	c.DB.DB().Where("1 = 1").Delete(models.Container{})
	c.DB.DB().Where("1 = 1").Delete(models.Workload{})

	if err := c.Agents.ExecuteInEachCluster(context.Background(), func(ctx context.Context, cli agents.Client) error {
		nsList := v1.NamespaceList{}
		if err := cli.List(ctx, &nsList, client.HasLabels([]string{gems.LabelEnvironment})); err != nil {
			return err
		}

		for _, ns := range nsList.Items {
			cpuPercentResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(containerCPUPercent_LastWeek, ns.Name))
			if err != nil {
				return err
			}
			memoryPercentResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(containerMemoryPercent_LastWeek, ns.Name))
			if err != nil {
				return err
			}
			cpuUsageResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(containerCPUUsageCore_LastWeek, ns.Name))
			if err != nil {
				return err
			}
			memoryUsageResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(containerMemoryUsageBytes_LastWeek, ns.Name))
			if err != nil {
				return err
			}
			// 由于是当前的cpu、内存限制，与上面的数据有出入
			cpuLimitResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(containerCPULimitCore, ns.Name))
			if err != nil {
				return err
			}
			memoryLimitResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(containerMemoryLimitBytes, ns.Name))
			if err != nil {
				return err
			}
			cpuLimitStdvarResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(workloadCPULimitStdVar, ns.Name))
			if err != nil {
				return err
			}
			memoryLimitStdvarResp, err := cli.Extend().PrometheusVector(ctx, fmt.Sprintf(workloadMemoryLimitStdVar, ns.Name))
			if err != nil {
				return err
			}

			// 缓存这个集群要插入的workload实例
			containerMap := make(map[string]*models.Container)
			// CPU使用率
			for _, sample := range cpuPercentResp {
				key, err := GetUniqueContainerKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				c := &models.Container{
					Workload: &models.Workload{
						ClusterName: cli.Name(),
						Namespace:   string(sample.Metric[NamespaceKey]),
						Type:        string(sample.Metric[WorkloadTypeKey]),
						Name:        strings.Split(string(sample.Metric[WorkloadNameKey]), ":")[1], // eg. Deployment:nginx
					},
					Name:       string(sample.Metric[ContainerKey]),
					PodName:    string(sample.Metric[PodKey]),
					CPUPercent: float64(sample.Value),
				}
				containerMap[key] = c
			}

			// 内存使用率
			for _, sample := range memoryPercentResp {
				key, err := GetUniqueContainerKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				if c, ok := containerMap[key]; ok {
					c.MemoryPercent = float64(sample.Value)
					containerMap[key] = c
				} else {
					c := &models.Container{
						Workload: &models.Workload{
							ClusterName: cli.Name(),
							Namespace:   string(sample.Metric[NamespaceKey]),
							Type:        string(sample.Metric[WorkloadTypeKey]),
							Name:        strings.Split(string(sample.Metric[WorkloadNameKey]), ":")[1],
						},
						Name:          string(sample.Metric[ContainerKey]),
						PodName:       string(sample.Metric[PodKey]),
						MemoryPercent: float64(sample.Value),
					}
					containerMap[key] = c
				}
			}

			// CPU使用量，在这之前所有超标的容器信息全部缓存完毕
			for _, sample := range cpuUsageResp {
				key, err := GetUniqueContainerKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				if c, ok := containerMap[key]; ok {
					c.CPUUsageCore = float64(sample.Value)
					containerMap[key] = c
				}
			}

			// 内存使用量
			for _, sample := range memoryUsageResp {
				key, err := GetUniqueContainerKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				if c, ok := containerMap[key]; ok {
					c.MemoryUsageBytes = float64(sample.Value)
					containerMap[key] = c
				}
			}

			// CPU限制
			for _, sample := range cpuLimitResp {
				key, err := GetUniqueContainerKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				if c, ok := containerMap[key]; ok {
					c.CPULimitCore = float64(sample.Value)
					containerMap[key] = c
				}
			}

			// 内存限制
			for _, sample := range memoryLimitResp {
				key, err := GetUniqueContainerKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				if c, ok := containerMap[key]; ok {
					c.MemoryLimitBytes = int64(sample.Value)
					containerMap[key] = c
				}
			}

			// 逆转containerMap
			workloadMap := make(map[string]*models.Workload)
			for cKey := range containerMap {
				wKey := containerMap[cKey].Workload.UniqueKey()
				w, ok := workloadMap[wKey]
				if !ok {
					// 获取container
					c := containerMap[cKey]
					w = c.Workload
					c.Workload = nil // 置为空，否则gorm关联插入会出问题

					w.Containers = append(w.Containers, c)
					workloadMap[wKey] = w
				} else {
					c := containerMap[cKey]
					c.Workload = nil

					w.Containers = append(w.Containers, c)
					workloadMap[wKey] = w
				}
			}

			// cpulimit方差
			for _, sample := range cpuLimitStdvarResp {
				key, err := GetUniqueWorkloadKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				if w, ok := workloadMap[key]; ok {
					w.CPULimitStdvar = float64(sample.Value)
					workloadMap[key] = w
				}
			}
			// 内存limit方差
			for _, sample := range memoryLimitStdvarResp {
				key, err := GetUniqueWorkloadKey(sample)
				if err != nil {
					log.Warnf("notvalid workload in cluster: %s, err:%v", cli.Name(), err)
					continue
				}
				if w, ok := workloadMap[key]; ok {
					w.MemoryLimitStdvar = float64(sample.Value)
					workloadMap[key] = w
				}
			}

			total := len(workloadMap)
			if total == 0 {
				continue
			}

			workloads := make([]models.Workload, total)
			index := 0
			for _, v := range workloadMap {
				workloads[index] = *v
				index++
			}
			if err := c.DB.DB().Save(&workloads).Error; err != nil {
				return errors.Wrap(err, "failed to save workload resources")
			}
			log.Infof("cluster %s, namespace: %s workload collect succeed, total: %d", cli.Name(), ns, total)
		}
		return nil
	}); err != nil {
		return err
	}

	log.Info("finish workload resource list", "duration", time.Since(start).String())
	return nil
}

// 通过 namespace, pod, container 生成唯一Key
func GetUniqueContainerKey(sample *promemodel.Sample) (string, error) {
	// 由namespace+podName+containerName 生成唯一key，其余的只做检查
	if sample.Metric[NamespaceKey] == "" {
		return "", fmt.Errorf("namespace key not found: %v", sample.Metric)
	}

	if sample.Metric[WorkloadTypeKey] != Deployment && sample.Metric[WorkloadTypeKey] != StatefulSet && sample.Metric[WorkloadTypeKey] != DaemonSet {
		return "", fmt.Errorf("owner_kind key not valid: %v", sample.Metric)
	}

	tmp := strings.Split(string(sample.Metric[WorkloadNameKey]), ":")
	if len(tmp) != 2 {
		return "", fmt.Errorf("workload key not valid: %v", sample.Metric)
	}

	if sample.Metric[PodKey] == "" {
		return "", fmt.Errorf("pod key not found: %v", sample.Metric)
	}

	if sample.Metric[ContainerKey] == "" {
		return "", fmt.Errorf("container key not found: %v", sample.Metric)
	}

	if math.IsInf(float64(sample.Value), 0) || math.IsNaN(float64(sample.Value)) {
		return "", fmt.Errorf("value not valid: %v", sample.Value)
	}

	return fmt.Sprintf("%s_%s_%s", sample.Metric[NamespaceKey], sample.Metric[PodKey], sample.Metric[ContainerKey]), nil
}

// 通过 namespace, owner_kind, workload 生成唯一Key
func GetUniqueWorkloadKey(sample *promemodel.Sample) (string, error) {
	// 由namespace+podName+containerName 生成唯一key，其余的只做检查
	if sample.Metric[NamespaceKey] == "" {
		return "", fmt.Errorf("namespace key not found: %v", sample.Metric)
	}

	if sample.Metric[WorkloadTypeKey] != Deployment && sample.Metric[WorkloadTypeKey] != StatefulSet && sample.Metric[WorkloadTypeKey] != DaemonSet {
		return "", fmt.Errorf("owner_kind key not valid: %v", sample.Metric)
	}

	tmp := strings.Split(string(sample.Metric[WorkloadNameKey]), ":")
	if len(tmp) != 2 {
		return "", fmt.Errorf("workload key not valid: %v", sample.Metric)
	}

	if math.IsInf(float64(sample.Value), 0) || math.IsNaN(float64(sample.Value)) {
		return "", fmt.Errorf("value not valid: %v", sample.Value)
	}

	return fmt.Sprintf("%s_%s_%s", sample.Metric[NamespaceKey], sample.Metric[WorkloadTypeKey], tmp[1]), nil
}
