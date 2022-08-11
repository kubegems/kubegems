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

package prometheus

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
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
	for _, res := range opts.Resources {
		for _, rule := range res.Rules {
			if _, err := ParseUnit(rule.Unit); err != nil {
				return errors.Wrapf(err, "res: %s, rule: %s unit not valid", res.ShowName, rule.ShowName)
			}
			_, err := promql.New(rule.Expr)
			if err != nil {
				return errors.Wrapf(err, "res: %s, rule: %s", res.ShowName, rule.ShowName)
			}
		}
	}
	return nil
}

func (opts *MonitorOptions) JSON() []byte {
	if err := opts.Validate(); err != nil {
		panic(err)
	}
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
	Unit     string   `json:"unit"`     // 使用的单位
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
        unit: percent-0-100
      memoryUsagePercent:
        expr: (1- sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes)) * 100
        showName: "内存使用率"
        labels: null
        unit: percent-0-100
      certExpirationRemainTime:
        expr: gems_agent_cluster_component_cert_expiration_remain_seconds
        showName: "证书剩余到期时间"
        labels: [component]
        unit: duration-s
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
        unit: short
      cpuUsage:
        expr: gems_node_cpu_usage_cores
        showName: "CPU使用量"
        labels: [node]
        unit: short
      cpuUsagePercent:
        expr: gems_node_cpu_usage_percent
        showName: "CPU使用率"
        labels: [node]
        unit: percent-0-100

      memoryTotal:
        expr: gems_node_memory_total_bytes
        showName: "内存总量"
        labels: [node]
        unit: bytes-B
      memoryUsage:
        expr: gems_node_memory_usage_bytes
        showName: "内存使用量"
        labels: [node]
        unit: bytes-B
      memoryUsagePercent:
        expr: gems_node_memory_usage_percent
        showName: "内存使用率"
        labels: [node]
        unit: percent-0-100

      diskTotal:
        expr: gems_node_disk_total_bytes
        showName: "磁盘总量"
        labels: [node, device]
        unit: bytes-B
      diskUsage:
        expr: gems_node_disk_usage_bytes
        showName: "磁盘使用量"
        labels: [node, device]
        unit: bytes-B
      diskUsagePercent:
        expr: gems_node_disk_usage_percent
        showName: "磁盘使用率"
        labels: [node, device]
        unit: percent-0-100
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
        unit: bytes/sec-B/s
      diskWriteBPS:
        expr: gems_node_disk_write_bps
        showName: "磁盘每秒写入量"
        labels: [node]
        unit: bytes/sec-B/s

      networkInBPS:
        expr: gems_node_network_receive_bps
        showName: "网络每秒接收流量"
        labels: [node]
        unit: bytes/sec-B/s
      networkOutBPS:
        expr: gems_node_network_send_bps
        showName: "网络每秒发送流量"
        labels: [node]
        unit: bytes/sec-B/s
      networkInErrPercent:
        expr: gems_node_network_receive_errs_percent
        showName: "网络接口收包错误率"
        labels: [node, instance, device]
        unit: percent-0-100
      networkOutErrPercent:
        expr: gems_node_network_send_errs_percent
        showName: "网络接口发包错误率"
        labels: [node, instance, device]
        unit: percent-0-100

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
        unit: percent-0-100

  container:
    namespaced: true
    showName: "容器"
    rules:
      cpuTotal:
        expr: gems_container_cpu_limit_cores
        showName: "CPU总量"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: short
      cpuUsage:
        expr: gems_container_cpu_usage_cores
        showName: "CPU使用量"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: short
      cpuUsagePercent:
        expr: gems_container_cpu_usage_percent
        showName: "CPU使用率"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: percent-0-100

      memoryTotal:
        expr: gems_container_memory_limit_bytes
        showName: "内存总量"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: bytes-B
      memoryUsage:
        expr: gems_container_memory_usage_bytes
        showName: "内存使用量"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: bytes-B
      memoryUsagePercent:
        expr: gems_container_memory_usage_percent
        showName: "内存使用率"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: percent-0-100

      networkInBPS:
        expr: gems_container_network_receive_bps
        showName: "网络每秒接收流量"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: bytes/sec-B/s
      networkOutBPS:
        expr: gems_container_network_send_bps
        showName: "网络每秒发送流量"
        labels: [node, namespace, pod, container, owner_kind, workload]
        unit: bytes/sec-B/s

      restartTimesLast5m:
        expr: gems_container_restart_times_last_5m
        showName: "过去5m重启次数"
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
        unit: bytes-B
      volumeUsage:
        expr: gems_pvc_usage_bytes
        showName: "存储卷使用量"
        labels: [node, namespace, persistentvolumeclaim]
        unit: bytes-B
      volumeUsagePercent:
        expr: gems_pvc_usage_percent
        showName: "存储卷使用率"
        labels: [node, namespace, persistentvolumeclaim]
        unit: percent-0-100
  cert:
    namespaced: true
    showName: "证书"
    rules:
      expirationRemainTime:
        expr: gems_cert_expiration_remain_seconds
        showName: "剩余到期时间"
        labels: [namespace, name]
        unit: duration-s
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
        unit: short
      memoryUsage:
        expr: gems_namespace_memory_usage_bytes
        showName: "内存使用量"
        labels: [tenant, project, environment, namespace]
        unit: bytes-B
      networkInBPS:
        expr: gems_namespace_network_receive_bps
        showName: "网络每秒接收流量"
        labels: [tenant, project, environment, namespace]
        unit: bytes/sec-B/s
      networkOutBPS:
        expr: gems_namespace_network_send_bps
        showName: "网络每秒发送流量"
        labels: [tenant, project, environment, namespace]
        unit: bytes/sec-B/s
      volumeUsage:
        expr: gems_namespace_pvc_usage_bytes
        showName: "存储卷使用量"
        labels: [tenant, project, environment, namespace]
        unit: bytes-B
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
  mysql:
    namespaced: true
    showName: mysql
    rules:
      mysqlQPS:
        expr: rate(mysql_global_status_queries[5m])
        unit: short
        labels:
          - service
        showName: MySQL实时QPS
      mysqlTPS:
        expr:
          "sum(rate(mysql_global_status_commands_total{command=~\"insert|update|delete\"}[5m]))
          without (command)\t"
        unit: short
        labels:
          - service
        showName: MySQL实时TPS
      mysqlState:
        expr: mysql_up
        unit: short
        labels:
          - service
        showName: MySQL状态
      mysqlThreads:
        expr: mysql_info_schema_threads
        unit: short
        labels:
          - service
          - state
        showName: MySQL线程数(by state)
      mysqlOpenFiles:
        expr: mysql_global_status_innodb_num_open_files
        unit: short
        labels:
          - service
        showName: MySQL打开文件数
      mysqlQuestions:
        expr: rate(mysql_global_status_questions[5m])
        unit: short
        labels:
          - service
        showName: MySQL查询速率(questions/s)
      mysqlSentBytes:
        expr: " irate(mysql_global_status_bytes_sent[5m])"
        unit: bytes/sec-B/s
        labels:
          - service
        showName: MySQL出口流量(bytes/s)
      mysqlSlowQuery:
        expr: idelta(mysql_global_status_slow_queries[5m])
        unit: short
        labels:
          - service
        showName: MySQL慢查询(slow_queries/s)
      mysqlTableSize:
        expr: sum by (schema) (mysql_info_schema_table_size)
        unit: bytes-B
        labels:
          - service
        showName: MySQL容量(bytes)
      mysqlTmpTables:
        expr: sum(rate(mysql_global_status_created_tmp_tables[5m]))
        unit: short
        labels:
          - service
        showName: MySQL创建临时表速率(tables/s)
      mysqlTotalRows:
        expr: sum(mysql_info_schema_table_rows)
        unit: short
        labels:
          - service
        showName: MySQL总数据(rows)
      mysqlConnections:
        expr: mysql_global_status_max_used_connections
        unit: short
        labels:
          - service
        showName: MySQL连接数
      mysqlCommandTop10:
        expr: topk(10, rate(mysql_global_status_commands_total[5m])>0)
        unit: short
        labels:
          - service
          - command
        showName: MySQL Top10命令
      mysqlReceivedBytes:
        expr: irate(mysql_global_status_bytes_received[5m])
        unit: bytes/sec-B/s
        labels:
          - service
        showName: MySQL入口流量(bytes/s)
      mysqlTabelLockWaited:
        expr: sum(increase(mysql_global_status_table_locks_waited[5m]))
        unit: short
        labels:
          - service
        showName: MySQL锁表等待(5m)
      mysqlInnodbBufferSize:
        expr: mysql_global_variables_innodb_buffer_pool_size
        unit: bytes-B
        labels:
          - service
        showName: MySQL Innodb缓冲区(Bytes)
      mysqlTableOpenCacheHitRatio:
        expr:
          rate(mysql_global_status_table_open_cache_hits[5m]) / ( rate(mysql_global_status_table_open_cache_hits[5m])
          + rate(mysql_global_status_table_open_cache_misses[5m]) )
        unit: percent-0.0-1.0
        labels:
          - service
        showName: MySQL表缓存命中率(%)
  redis:
    namespaced: true
    showName: redis
    rules:
      redisOPS:
        expr: irate(redis_commands_processed_total[5m])
        unit: short
        labels:
          - service
        showName: Redis每秒操作(op/s)
      redisKeys:
        expr: sum (redis_db_keys) by (db)
        unit: short
        labels:
          - service
          - db
        showName: Redis Keys(by dbs)
      redisState:
        expr: redis_up
        unit: short
        labels:
          - service
        showName: Redis 状态
      redisClients:
        expr: redis_connected_clients
        unit: short
        labels:
          - service
        showName: Redis 客户端
      redisCpuUsage:
        expr: "irate(redis_cpu_user_seconds_total[5m]) + irate(redis_cpu_sys_seconds_total[5m])\t"
        unit: short
        labels:
          - service
        showName: Redis CPU使用量(1000m=1Core)
      redisKeysHits:
        expr: redis_keyspace_hits_total
        unit: short
        labels:
          - service
        showName: Redis Keys总命中
      redisSentBytes:
        expr: irate(redis_net_output_bytes_total[5m])
        unit: bytes/sec-B/s
        labels:
          - service
        showName: Redis出口流量(bytes/s)
      redisKeysMissed:
        expr: redis_keyspace_misses_total
        unit: short
        labels:
          - service
        showName: Redis Keys总未命中
      redisKeysEvicted:
        expr: redis_evicted_keys_total
        unit: short
        labels:
          - service
        showName: Redis Evicted Keys
      redisKeysHitRate:
        expr: redis_keyspace_hits_total / (redis_keyspace_hits_total+ redis_keyspace_misses_total)
        unit: percent-0-100
        labels:
          - service
        showName: Redis命中率(%)
      redisCommandsTop5:
        expr: topk(5, irate(redis_commands_total[1m]))
        unit: short
        labels:
          - service
          - cmd
        showName: Redis命令Top5
      redisKeysExpiring:
        expr: redis_db_keys_expiring
        unit: short
        labels:
          - service
        showName: Redis Expireing Keys
      redisReceivedByted:
        expr: irate(redis_net_input_bytes_total[5m])
        unit: bytes/sec-B/s
        labels:
          - service
        showName: Redis入口流量(bytes/s)
      redisMemoryUsedBytes:
        expr: redis_memory_used_bytes
        unit: bytes-B
        labels:
          - service
        showName: Redis内存占用(bytes)
      redisNoExpiringKeyRate:
        expr: 1 - sum(redis_db_keys_expiring) / sum(redis_db_keys)
        unit: percent-0.0-1.0
        labels:
          - service
        showName: Redis永久Key比例(%)
      redisRejectedConnectionsTotal:
        expr: redis_rejected_connections_total
        unit: short
        labels:
          - service
        showName: Redis拒绝连接总数
  mongodb:
    namespaced: true
    showName: mongodb
    rules:
      mongodbQPS:
        expr: "sum(rate(mongodb_op_counters_total{type=~\"query|getmore\"}[5m]))\t"
        unit: short
        labels:
          - service
        showName: MongoDB QPS
      mongodbTPS:
        expr: sum(rate(mongodb_op_counters_total{type=~"insert|update|delete"}[5m]))
        unit: short
        labels:
          - service
        showName: MongoDB TPS
      mongdbState:
        expr: mongodb_up
        unit: short
        labels:
          - service
        showName: MongoDB状态
      mongodbCursor:
        expr: mongodb_mongod_metrics_cursor_open
        unit: short
        labels:
          - service
          - state
        showName: MongoDB游标数量
      mongodbMemory:
        expr: mongodb_memory
        unit: bytes-MB
        labels:
          - service
          - type
        showName: MongoDB内存使用量(M Bytes)
      mongodbAsserts:
        expr: rate(mongodb_asserts_total[5m])
        unit: short
        labels:
          - service
          - type
        showName: MongoDB断言错误次数
      mongodbObjects:
        expr: mongodb_mongod_db_objects_total
        unit: short
        labels:
          - service
          - db
        showName: MongoDB对象数
      mongodbDataSize:
        expr: mongodb_mongod_db_data_size_bytes
        unit: bytes-B
        labels:
          - service
          - db
        showName: MongoDB数据容量(bytes)
      mongodbDocument:
        expr: mongodb_mongod_metrics_document_total
        unit: short
        labels:
          - service
          - state
        showName: MongoDB文档操作数(op/s)
      mongodbIndexSize:
        expr: mongodb_mongod_db_index_size_bytes
        unit: bytes-B
        labels:
          - service
          - db
        showName: MongoDB索引容量(bytes)
      mongodbLockQueue:
        expr: mongodb_mongod_global_lock_current_queue
        unit: short
        labels:
          - service
          - type
        showName: MongoDB等待获取锁操作数量
      mongodbOplogSize:
        expr: mongodb_mongod_replset_oplog_size_bytes
        unit: bytes-B
        labels:
          - service
          - type
        showName: MongoDB Oplog容量(bytes)
      mongodbSentBytes:
        expr: mongodb_network_bytes_total{state="out_bytes"}
        unit: bytes/sec-B/s
        labels:
          - service
        showName: MongoDB出口流量(bytes/s)
      mongodbCacheBytes:
        expr: mongodb_mongod_wiredtiger_cache_bytes
        unit: bytes-B
        labels:
          - service
          - type
        showName: MongoDB缓存容量(Bytes)
      mongodbGlobalLock:
        expr: mongodb_mongod_global_lock_total
        unit: short
        labels:
          - service
        showName: MongoDB全局锁
      mongodbPageFaults:
        expr: mongodb_extra_info_page_faults_total
        unit: short
        labels:
          - service
        showName: MongoDB页缺失中断次数
      mongdoResponseTime:
        expr: "rate(mongodb_mongod_op_latencies_latency_total[5m]) / rate(mongodb_mongod_op_latencies_ops_total[5m]) "
        unit: duration-ms
        labels:
          - service
          - type
        showName: MongoDB操作详情耗时(ms)
      mongodbConnections:
        expr: mongodb_connections
        unit: short
        labels:
          - service
          - state
        showName: MongoDB 连接数
      mongodbReceivedBytes:
        expr: mongodb_network_bytes_total{state="in_bytes"}
        unit: bytes/sec-B/s
        labels:
          - service
        showName: MongoDB入口流量(bytes/s)
      mongodbWiredtigerCacheRate:
        expr: sum(mongodb_mongod_wiredtiger_cache_bytes{type="total"}) / sum(mongodb_mongod_wiredtiger_cache_bytes_total)
        unit: percent-0-100
        labels:
          - service
        showName: MongoDB缓存使用率(%)
  kafka:
    namespaced: true
    showName: kafka
    rules:
      kafkaTopics:
        expr: count(count by (topic) (kafka_topic_partitions))
        unit: short
        labels:
          - service
        showName: Kafka Topics
      kafkaBrokers:
        expr: kafka_brokers
        unit: short
        labels:
          - service
        showName: Kafka Brokers
      kafkaPartitions:
        expr: sum(kafka_topic_partitions)
        unit: short
        labels:
          - service
        showName: Kafka Partitions
      kafkaConsumerLatency:
        expr: sum by (consumergroup,topic) (kafka_consumergroup_lag)
        unit: short
        labels:
          - service
          - " consumergroup"
          - topic
        showName: Kafka消息延迟
      kafkaMessageConsumer:
        expr: sum(rate(kafka_consumergroup_current_offset[1m])) by (topic)
        unit: short
        labels:
          - service
          - topic
        showName: Kafka消息消费(by topic)
      kafkaMessagesProduced:
        expr: sum(rate(kafka_topic_partition_current_offset[1m])) by (topic)
        unit: short
        labels:
          - service
          - topic
        showName: Kafka消息生产(by Topic)            
`)
	opts := &MonitorOptions{}
	if err := yaml.Unmarshal(bts, opts); err != nil {
		log.Error(err, "unmarshal monitor config")
	}
	return opts
}
