package prometheus

import (
	"encoding/json"
	"fmt"

	gemlabels "kubegems.io/pkg/labels"
	"kubegems.io/pkg/utils"
)

var (
	// 分开存放，避免map值被修改
	adminGlobalMetricCfg  GemsMetricConfig
	normalGlobalMetricCfg GemsMetricConfig
)

const (
	// 全局告警命名空间，非此命名空间强制加上namespace筛选
	GlobalAlertNamespace = gemlabels.NamespaceMonitor
	// namespace
	PromqlNamespaceKey = "namespace"
	// 配置路径写死
	configPath = "config/metricconfig.yaml"
)

func GetGemsMetricConfig(adminConfig bool) GemsMetricConfig {
	if adminConfig {
		return adminGlobalMetricCfg
	}
	return normalGlobalMetricCfg
}

func setNormalAndAdminConfig(cfg GemsMetricConfig) {
	adminGlobalMetricCfg = cfg
	normalGlobalMetricCfg = GemsMetricConfig{
		Units:     adminGlobalMetricCfg.Units,
		Severity:  adminGlobalMetricCfg.Severity,
		Operators: adminGlobalMetricCfg.Operators,
		Resources: make(map[string]ResourceDetail),
	}

	for resname, res := range adminGlobalMetricCfg.Resources {
		newDetail := ResourceDetail{
			Namespaced: res.Namespaced,
			ShowName:   res.ShowName,
			Rules:      make(map[string]RuleDetail),
		}
		if res.Namespaced {
			for rulename, rule := range res.Rules {
				rule.Labels = utils.RemoveStr(rule.Labels, PromqlNamespaceKey)
				newDetail.Rules[rulename] = rule
			}
			normalGlobalMetricCfg.Resources[resname] = newDetail
		}
	}
}

func (cfg GemsMetricConfig) CheckSelf() error {
	for _, res := range adminGlobalMetricCfg.Resources {
		for _, rule := range res.Rules {
			for _, unit := range rule.Units {
				if _, ok := adminGlobalMetricCfg.Units[unit]; !ok {
					return fmt.Errorf(fmt.Sprintf("unit %s not defind", unit))
				}
			}
		}
	}
	return nil
}

type ResourceDetail struct {
	Namespaced bool                  `json:"namespaced"` // 是否带有namespace
	ShowName   string                `json:"showName"`
	Rules      map[string]RuleDetail `json:"rules"`
}

type RuleDetail struct {
	Expr     string   `json:"expr"`     // 原生表达式
	ShowName string   `json:"showName"` // 前端展示
	Units    []string `json:"units"`    // 支持的单位
	Labels   []string `json:"labels"`   // 支持的标签
}

type GemsMetricConfig struct {
	Units     map[string]string `json:"units"`     // 单位
	Severity  map[string]string `json:"severity"`  // 告警级别
	Operators []string          `json:"operators"` // 运算符

	Resources map[string]ResourceDetail `json:"resources"` // 告警列表
}

func (cfg GemsMetricConfig) Reload() error {
	if err := cfg.CheckSelf(); err != nil {
		return err
	}
	adminGlobalMetricCfg = cfg
	setNormalAndAdminConfig(cfg)
	return nil
}

func DefaultMetricConfigContent() GemsMetricConfig {
	bts := []byte(`{
		"units": {
		  "percent": "%",
		  "core": "核",
		  "mcore": "毫核",
		  "b": "B",
		  "kb": "KB",
		  "mb": "MB",
		  "gb": "GB",
		  "tb": "TB",
		  "bps": "B/s",
		  "kbps": "KB/s",
		  "mbps": "MB/s",
		  "ops": "次/s",
		  "count": "个",
		  "times": "次",
		  "us": "微秒",
		  "ms": "毫秒",
		  "s": "秒",
		  "m": "分钟",
		  "h": "小时",
		  "d": "天",
		  "w": "周"
		},
		"severity": {
		  "error": "错误",
		  "critical": "严重"
		},
		"operators": [
		  "==",
		  "!=",
		  ">",
		  "<",
		  ">=",
		  "<="
		],
		"resources": {
		  "cluster": {
			"namespaced": false,
			"showName": "集群",
			"rules": {
			  "cpuUsagePercent": {
				"expr": "(1 - avg(irate(node_cpu_seconds_total{mode=\"idle\"}[5m]))) * 100",
				"showName": "CPU使用率",
				"units": [
				  "percent"
				],
				"labels": null
			  },
			  "memoryUsagePercent": {
				"expr": "(1- sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes)) * 100",
				"showName": "内存使用率",
				"units": [
				  "percent"
				],
				"labels": null
			  }
			}
		  },
		  "plugin": {
			"namespaced": false,
			"showName": "插件",
			"rules": {
			  "status": {
				"expr": "gems_server_plugin_status",
				"showName": "状态",
				"unit": null,
				"labels": [
				  "type",
				  "namespace",
				  "plugin",
				  "version",
				  "enabled"
				]
			  }
			}
		  },
		  "exporter": {
			"namespaced": false,
			"showName": "采集器",
			"rules": {
			  "status": {
				"expr": "up",
				"showName": "状态",
				"unit": null,
				"labels": [
				  "instance",
				  "job"
				]
			  }
			}
		  },
		  "node": {
			"namespaced": false,
			"showName": "节点",
			"rules": {
			  "cpuTotal": {
				"expr": "gems_node_cpu_total_cores",
				"showName": "CPU总量",
				"units": [
				  "core",
				  "mcore"
				],
				"labels": [
				  "host"
				]
			  },
			  "cpuUsage": {
				"expr": "gems_node_cpu_usage_cores",
				"showName": "CPU使用量",
				"units": [
				  "core",
				  "mcore"
				],
				"labels": [
				  "host"
				]
			  },
			  "cpuUsagePercent": {
				"expr": "gems_node_cpu_usage_percent",
				"showName": "CPU使用率",
				"units": [
				  "percent"
				],
				"labels": [
				  "host"
				]
			  },
			  "memoryTotal": {
				"expr": "gems_node_memory_total_bytes",
				"showName": "内存总量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "host"
				]
			  },
			  "memoryUsage": {
				"expr": "gems_node_memory_usage_bytes",
				"showName": "内存使用量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "host"
				]
			  },
			  "memoryUsagePercent": {
				"expr": "gems_node_memory_usage_percent",
				"showName": "内存使用率",
				"units": [
				  "percent"
				],
				"labels": [
				  "host"
				]
			  },
			  "diskTotal": {
				"expr": "gems_node_disk_total_bytes",
				"showName": "磁盘总量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "host",
				  "device"
				]
			  },
			  "diskUsage": {
				"expr": "gems_node_disk_usage_bytes",
				"showName": "磁盘使用量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "host",
				  "device"
				]
			  },
			  "diskUsagePercent": {
				"expr": "gems_node_disk_usage_percent",
				"showName": "磁盘使用率",
				"units": [
				  "percent"
				],
				"labels": [
				  "host",
				  "device"
				]
			  },
			  "diskReadIOPS": {
				"expr": "gems_node_disk_read_iops",
				"showName": "磁盘每秒读取次数",
				"units": [
				  "ops"
				],
				"labels": [
				  "host"
				]
			  },
			  "diskWriteIOPS": {
				"expr": "gems_node_disk_write_iops",
				"showName": "磁盘每秒写入次数",
				"units": [
				  "ops"
				],
				"labels": [
				  "host"
				]
			  },
			  "diskReadBPS": {
				"expr": "gems_node_disk_read_bps",
				"showName": "磁盘每秒读取量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "host"
				]
			  },
			  "diskWriteBPS": {
				"expr": "gems_node_disk_write_bps",
				"showName": "磁盘每秒写入量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "host"
				]
			  },
			  "networkInBPS": {
				"expr": "gems_node_network_receive_bps",
				"showName": "网络每秒接收流量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "host"
				]
			  },
			  "networkOutBPS": {
				"expr": "gems_node_network_send_bps",
				"showName": "网络每秒发送流量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "host"
				]
			  },
			  "networkInErrPercent": {
				"expr": "gems_node_network_receive_errs_percent",
				"showName": "网络接口收包错误率",
				"units": [
				  "percent"
				],
				"labels": [
				  "host",
				  "instance",
				  "device"
				]
			  },
			  "networkOutErrPercent": {
				"expr": "gems_node_network_send_errs_percent",
				"showName": "网络接口发包错误率",
				"units": [
				  "percent"
				],
				"labels": [
				  "host",
				  "instance",
				  "device"
				]
			  },
			  "load1": {
				"expr": "gems_node_load1",
				"showName": "最近1分钟平均负载",
				"units": null,
				"labels": [
				  "host"
				]
			  },
			  "load5": {
				"expr": "gems_node_load5",
				"showName": "最近5分钟平均负载",
				"units": null,
				"labels": [
				  "host"
				]
			  },
			  "load15": {
				"expr": "gems_node_load15",
				"showName": "最近15分钟平均负载",
				"units": null,
				"labels": [
				  "host"
				]
			  },
			  "statusCondition": {
				"expr": "kube_node_status_condition",
				"showName": "状态",
				"units": null,
				"labels": [
				  "node",
				  "condition",
				  "status"
				]
			  },
			  "runningPodCount": {
				"expr": "gems_node_running_pod_count",
				"showName": "运行中的pod数",
				"units": [
				  "count"
				],
				"labels": [
				  "node"
				]
			  },
			  "runningPodPercent": {
				"expr": "gems_node_running_pod_percent",
				"showName": "pod使用率",
				"units": [
				  "percent"
				],
				"labels": [
				  "node"
				]
			  }
			}
		  },
		  "container": {
			"namespaced": true,
			"showName": "容器",
			"rules": {
			  "cpuTotal": {
				"expr": "gems_container_cpu_limit_cores",
				"showName": "CPU总量",
				"units": [
				  "core",
				  "mcore"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "cpuUsage": {
				"expr": "gems_container_cpu_usage_cores",
				"showName": "CPU使用量",
				"units": [
				  "core",
				  "mcore"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "cpuUsagePercent": {
				"expr": "gems_container_cpu_usage_percent",
				"showName": "CPU使用率",
				"units": [
				  "percent"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "memoryTotal": {
				"expr": "gems_container_memory_limit_bytes",
				"showName": "内存总量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "memoryUsage": {
				"expr": "gems_container_memory_usage_bytes",
				"showName": "内存使用量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "memoryUsagePercent": {
				"expr": "gems_container_memory_usage_percent",
				"showName": "内存使用率",
				"units": [
				  "percent"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "networkInBPS": {
				"expr": "gems_container_network_receive_bps",
				"showName": "网络每秒接收流量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "networkOutBPS": {
				"expr": "gems_container_network_send_bps",
				"showName": "网络每秒发送流量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "node",
				  "namespace",
				  "pod",
				  "container",
				  "owner_kind",
				  "workload"
				]
			  },
			  "restartTimesLast5m": {
				"expr": "gems_container_restart_times_last_5m",
				"showName": "过去5m重启次数",
				"units": [
				  "times"
				],
				"labels": [
				  "namespace",
				  "pod",
				  "container"
				]
			  },
			  "statusTerminatedReason": {
				"expr": "kube_pod_container_status_terminated_reason",
				"showName": "终止原因",
				"units": null,
				"labels": [
				  "namespace",
				  "pod",
				  "container",
				  "reason"
				]
			  }
			}
		  },
		  "pvc": {
			"namespaced": true,
			"showName": "存储卷",
			"rules": {
			  "volumeTotal": {
				"expr": "gems_pvc_total_bytes",
				"showName": "存储卷容量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "node",
				  "namespace",
				  "persistentvolumeclaim"
				]
			  },
			  "volumeUsage": {
				"expr": "gems_pvc_usage_bytes",
				"showName": "存储卷使用量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "node",
				  "namespace",
				  "persistentvolumeclaim"
				]
			  },
			  "volumeUsagePercent": {
				"expr": "gems_pvc_usage_percent",
				"showName": "存储卷使用率",
				"units": [
				  "percent"
				],
				"labels": [
				  "node",
				  "namespace",
				  "persistentvolumeclaim"
				]
			  }
			}
		  },
		  "cert": {
			"namespaced": true,
			"showName": "证书",
			"rules": {
			  "expirationRemainTime": {
				"expr": "gems_cert_expiration_remain_seconds",
				"showName": "剩余到期时间",
				"units": [
				  "s",
				  "m",
				  "h",
				  "d",
				  "w"
				],
				"labels": [
				  "namespace",
				  "name"
				]
			  },
			  "status": {
				"expr": "certmanager_certificate_ready_status",
				"showName": "状态",
				"unit": null,
				"labels": [
				  "namespace",
				  "name",
				  "condition"
				]
			  }
			}
		  },
		  "environment": {
			"namespaced": true,
			"showName": "环境",
			"rules": {
			  "cpuUsage": {
				"expr": "gems_namespace_cpu_usage_cores",
				"showName": "CPU使用量",
				"units": [
				  "core",
				  "mcore"
				],
				"labels": [
				  "tenant",
				  "project",
				  "environment",
				  "namespace"
				]
			  },
			  "memoryUsage": {
				"expr": "gems_namespace_memory_usage_bytes",
				"showName": "内存使用量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "tenant",
				  "project",
				  "environment",
				  "namespace"
				]
			  },
			  "networkInBPS": {
				"expr": "gems_namespace_network_receive_bps",
				"showName": "网络每秒接收流量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "tenant",
				  "project",
				  "environment",
				  "namespace"
				]
			  },
			  "networkOutBPS": {
				"expr": "gems_namespace_network_send_bps",
				"showName": "网络每秒发送流量",
				"units": [
				  "bps",
				  "kbps",
				  "mbps"
				],
				"labels": [
				  "tenant",
				  "project",
				  "environment",
				  "namespace"
				]
			  },
			  "volumeUsage": {
				"expr": "gems_namespace_pvc_usage_bytes",
				"showName": "存储卷使用量",
				"units": [
				  "b",
				  "kb",
				  "mb",
				  "gb",
				  "tb"
				],
				"labels": [
				  "tenant",
				  "project",
				  "environment",
				  "namespace"
				]
			  }
			}
		  }
		}
	  }`)
	ret := GemsMetricConfig{}
	json.Unmarshal(bts, &ret)
	return ret
}
