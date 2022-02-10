package prometheus

import (
	"fmt"
	"io"
	"os"

	gemlabels "github.com/kubegems/gems/pkg/labels"
	"github.com/kubegems/gems/pkg/utils"
	"sigs.k8s.io/yaml"
)

// 除法单位表
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

func Init() {
	f, err := os.Open(configPath)
	if err != nil {
		panic(err)
	}

	bts, _ := io.ReadAll(f)
	if err := yaml.Unmarshal(bts, &adminGlobalMetricCfg); err != nil {
		panic(err)
	}

	if err := adminGlobalMetricCfg.CheckSelf(); err != nil {
		panic(err)
	}

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
