package prometheus

import (
	"encoding/json"
	"fmt"

	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"sigs.k8s.io/yaml"
)

const (
	// 全局告警命名空间，非此命名空间强制加上namespace筛选
	GlobalAlertNamespace = gemlabels.NamespaceMonitor
	// namespace
	PromqlNamespaceKey = "namespace"
)

func (opts *MonitorOptions) Name() string {
	return "Monitor"
}

func (opts *MonitorOptions) Validate() error {
	return nil
}

func (opts *MonitorOptions) JSON() []byte {
	bts, _ := json.Marshal(opts)
	return bts
}

func (cfg *MonitorOptions) FindRuleContext(resName, ruleName string) (RuleContext, error) {
	ctx := RuleContext{}
	resourceDetail, ok := cfg.Resources[resName]
	if !ok {
		return ctx, fmt.Errorf("invalid resource: %s", resName)
	}

	ruleDetail, ok := resourceDetail.Rules[ruleName]
	if !ok {
		return ctx, fmt.Errorf("rule %s not in resource %s", ruleName, resName)
	}

	ctx.ResourceDetail = resourceDetail
	ctx.RuleDetail = ruleDetail
	return ctx, nil
}

type ResourceDetail struct {
	Namespaced bool                  `json:"namespaced"` // 是否带有namespace
	ShowName   string                `json:"showName"`
	Rules      map[string]RuleDetail `json:"rules"`
}

type RuleDetail struct {
	Expr     string   `json:"expr"`     // 原生表达式
	ShowName string   `json:"showName"` // 前端展示
	Labels   []string `json:"labels"`   // 支持的标签
}

type MonitorOptions struct {
	Severity  map[string]string `json:"severity"`  // 告警级别
	Operators []string          `json:"operators"` // 运算符

	Resources map[string]ResourceDetail `json:"resources"` // 告警列表
}

func DefaultMonitorOptions() *MonitorOptions {
	bts := []byte(`
# 支持的告警等级
# 用户选择时，展示value，接口传key
severity:
  error: "错误"
  critical: "严重"

# 支持的运算符
operators: ["==", "!=", ">", "<", ">=", "<="]

# 支持的监控、告警指标
# 1. 选择资源类型(如node、container等, 前端展示对应的showName)
# 2. 选择指标: rules
# 3. 选择单位: units(非必须，默认填第一个)
# 4. 执行查询
resources:
  cluster:
    namespaced: false
    showName: "集群"
    rules:
      cpuUsagePercent:
        expr: (1 - avg(irate(node_cpu_seconds_total{mode="idle"}[5m]))) * 100
        showName: "CPU使用率"
        labels: null
      memoryUsagePercent:
        expr: (1- sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes)) * 100
        showName: "内存使用率"
        labels: null
      certExpirationRemainTime:
        expr: gems_agent_cluster_component_cert_expiration_remain_seconds
        showName: "证书剩余到期时间"
        labels: [component]
  plugin:
    namespaced: false
    showName: "插件"
    rules:
      status:
        expr: gems_server_plugin_status
        showName: "状态"
        labels: [type, namespace, plugin, version, enabled]
  exporter:
    namespaced: false
    showName: "采集器"
    rules:
      status:
        expr: up
        showName: "状态"
        labels: [instance, job]
  node: # 监控namespace才能使用
    namespaced: false
    showName: "节点"
    rules:
      cpuTotal:
        expr: gems_node_cpu_total_cores
        showName: "CPU总量"
        labels: [node]
      cpuUsage:
        expr: gems_node_cpu_usage_cores
        showName: "CPU使用量"
        labels: [node]
      cpuUsagePercent:
        expr: gems_node_cpu_usage_percent
        showName: "CPU使用率"
        labels: [node]

      memoryTotal:
        expr: gems_node_memory_total_bytes
        showName: "内存总量"
        labels: [node]
      memoryUsage:
        expr: gems_node_memory_usage_bytes
        showName: "内存使用量"
        labels: [node]
      memoryUsagePercent:
        expr: gems_node_memory_usage_percent
        showName: "内存使用率"
        labels: [node]

      diskTotal:
        expr: gems_node_disk_total_bytes
        showName: "磁盘总量"
        labels: [node, device]
      diskUsage:
        expr: gems_node_disk_usage_bytes
        showName: "磁盘使用量"
        labels: [node, device]
      diskUsagePercent:
        expr: gems_node_disk_usage_percent
        showName: "磁盘使用率"
        labels: [node, device]
      diskReadIOPS:
        expr: gems_node_disk_read_iops
        showName: "磁盘每秒读取次数"
        labels: [node]
      diskWriteIOPS:
        expr: gems_node_disk_write_iops
        showName: "磁盘每秒写入次数"
        labels: [node]
      diskReadBPS:
        expr: gems_node_disk_read_bps
        showName: "磁盘每秒读取量"
        labels: [node]
      diskWriteBPS:
        expr: gems_node_disk_write_bps
        showName: "磁盘每秒写入量"
        labels: [node]

      networkInBPS:
        expr: gems_node_network_receive_bps
        showName: "网络每秒接收流量"
        labels: [node]
      networkOutBPS:
        expr: gems_node_network_send_bps
        showName: "网络每秒发送流量"
        labels: [node]
      networkInErrPercent:
        expr: gems_node_network_receive_errs_percent
        showName: "网络接口收包错误率"
        labels: [node, instance, device]
      networkOutErrPercent:
        expr: gems_node_network_send_errs_percent
        showName: "网络接口发包错误率"
        labels: [node, instance, device]

      load1:
        expr: gems_node_load1
        showName: "最近1分钟平均负载"
        labels: [node]
      load5:
        expr: gems_node_load5
        showName: "最近5分钟平均负载"
        labels: [node]
      load15:
        expr: gems_node_load15
        showName: "最近15分钟平均负载"
        labels: [node]

      # k8s节点指标
      statusCondition:
        expr: kube_node_status_condition
        showName: "状态"
        labels: [node, condition, status]
      runningPodCount:
        expr: gems_node_running_pod_count
        showName: "运行中的pod数"
        labels: [node]
      runningPodPercent:
        expr: gems_node_running_pod_percent
        showName: "pod使用率"
        labels: [node]

  container:
    namespaced: true
    showName: "容器"
    rules:
      cpuTotal:
        expr: gems_container_cpu_limit_cores
        showName: "CPU总量"
        labels: [node, namespace, pod, container, owner_kind, workload]
      cpuUsage:
        expr: gems_container_cpu_usage_cores
        showName: "CPU使用量"
        labels: [node, namespace, pod, container, owner_kind, workload]
      cpuUsagePercent:
        expr: gems_container_cpu_usage_percent
        showName: "CPU使用率"
        labels: [node, namespace, pod, container, owner_kind, workload]

      memoryTotal:
        expr: gems_container_memory_limit_bytes
        showName: "内存总量"
        labels: [node, namespace, pod, container, owner_kind, workload]
      memoryUsage:
        expr: gems_container_memory_usage_bytes
        showName: "内存使用量"
        labels: [node, namespace, pod, container, owner_kind, workload]
      memoryUsagePercent:
        expr: gems_container_memory_usage_percent
        showName: "内存使用率"
        labels: [node, namespace, pod, container, owner_kind, workload]

      networkInBPS:
        expr: gems_container_network_receive_bps
        showName: "网络每秒接收流量"
        labels: [node, namespace, pod, container, owner_kind, workload]
      networkOutBPS:
        expr: gems_container_network_send_bps
        showName: "网络每秒发送流量"
        labels: [node, namespace, pod, container, owner_kind, workload]

      restartTimesLast5m:
        expr: gems_container_restart_times_last_5m
        showName: "过去5m重启次数"
        units: [times]
        labels: [namespace, pod, container]
      statusTerminatedReason:
        expr: kube_pod_container_status_terminated_reason
        showName: "终止原因"
        labels: [namespace, pod, container, reason]

  pvc:
    namespaced: true
    showName: "存储卷"
    rules:
      volumeTotal:
        expr: gems_pvc_total_bytes
        showName: "存储卷容量"
        labels: [node, namespace, persistentvolumeclaim]
      volumeUsage:
        expr: gems_pvc_usage_bytes
        showName: "存储卷使用量"
        labels: [node, namespace, persistentvolumeclaim]
      volumeUsagePercent:
        expr: gems_pvc_usage_percent
        showName: "存储卷使用率"
        labels: [node, namespace, persistentvolumeclaim]
  cert:
    namespaced: true
    showName: "证书"
    rules:
      expirationRemainTime:
        expr: gems_cert_expiration_remain_seconds
        showName: "剩余到期时间"
        labels: [namespace, name]
      status:
        expr: certmanager_certificate_ready_status
        showName: "状态"
        labels: [namespace, name, condition]
  environment:
    namespaced: true
    showName: "环境"
    rules:
      cpuUsage:
        expr: gems_namespace_cpu_usage_cores
        showName: "CPU使用量"
        labels: [tenant, project, environment, namespace]
      memoryUsage:
        expr: gems_namespace_memory_usage_bytes
        showName: "内存使用量"
        labels: [tenant, project, environment, namespace]
      networkInBPS:
        expr: gems_namespace_network_receive_bps
        showName: "网络每秒接收流量"
        labels: [tenant, project, environment, namespace]
      networkOutBPS:
        expr: gems_namespace_network_send_bps
        showName: "网络每秒发送流量"
        labels: [tenant, project, environment, namespace]
      volumeUsage:
        expr: gems_namespace_pvc_usage_bytes
        showName: "存储卷使用量"
        labels: [tenant, project, environment, namespace]
  log:
    namespaced: true
    showName: "日志"
    rules:
      logCount:
        expr: gems_loki_logs_count_last_1m
        showName: "过去一分钟日志行数"
        labels: [namespace, pod, container]
      errorLogCount:
        expr: gems_loki_error_logs_count_last_1m
        showName: "过去一分钟错误日志行数"
        labels: [namespace, pod, container]
`)
	opts := &MonitorOptions{}
	if err := yaml.Unmarshal(bts, opts); err != nil {
		log.Error(err, "unmarshal monitor config")
	}
	return opts
}
