package main

import (
	"fmt"
	"io/fs"
	"os"
	"sort"

	"kubegems.io/kubegems/pkg/service/models"
	"sigs.k8s.io/yaml"
)

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

func main() {
	old := MonitorOptions{}
	bts, _ := os.ReadFile("scripts/generate-promql-tpl/old.yaml")
	yaml.Unmarshal(bts, &old)

	tpl := []*models.PromqlTplScope{
		{
			ID:         1,
			Name:       "system",
			ShowName:   "系统资源",
			Namespaced: false,
			Resources: []*models.PromqlTplResource{
				{
					Name: "cluster",
				},
				{
					Name: "node",
				},
				{
					Name: "environment",
				},
			},
		},
		{
			ID:         2,
			Name:       "containers",
			ShowName:   "容器资源",
			Namespaced: true,
			Resources: []*models.PromqlTplResource{
				{
					Name: "container",
				},
				{
					Name: "pvc",
				},
				{
					Name: "log",
				},
			},
		},
		{
			ID:         3,
			Name:       "middlewires",
			ShowName:   "中间件",
			Namespaced: true,
			Resources: []*models.PromqlTplResource{
				{
					Name: "mysql",
				},
				{
					Name: "redis",
				},
				{
					Name: "kafka",
				},
				{
					Name: "mongodb",
				},
				{
					Name: "elasticsearch",
				},
				{
					Name: "postgres",
				},
				{
					Name: "consule",
				},
			},
		},
		{
			ID:         4,
			Name:       "others",
			ShowName:   "其他",
			Namespaced: true,
			Resources: []*models.PromqlTplResource{
				{
					Name: "cert",
				},
				{
					Name: "plugin",
				},
				{
					Name: "exporter",
				},
			},
		},
	}

	for _, scope := range tpl {
		for j, res := range scope.Resources {
			oldres, ok := old.Resources[res.Name]
			if !ok {
				panic(fmt.Errorf("res: %s not found", res.Name))
			}
			res.ScopeID = &scope.ID
			res.ShowName = oldres.ShowName
			for key, oldrule := range oldres.Rules {
				res.Rules = append(scope.Resources[j].Rules, &models.PromqlTplRule{
					ResourceID: &res.ID,
					Name:       key,
					ShowName:   oldrule.ShowName,
					Expr:       oldrule.Expr,
					Unit:       oldrule.Unit,
					Labels:     oldrule.Labels,
				})
			}
			sort.Slice(res.Rules, func(i, j int) bool {
				return res.Rules[i].Name < res.Rules[j].Name
			})
		}
		sort.Slice(scope.Resources, func(i, j int) bool {
			return scope.Resources[i].Name < scope.Resources[j].Name
		})
	}
	sort.Slice(tpl, func(i, j int) bool {
		return tpl[i].Name < tpl[j].Name
	})

	var resCount, ruleCount uint
	for _, scope := range tpl {
		for _, res := range scope.Resources {
			resCount += 1
			res.ID = resCount
			res.ScopeID = &scope.ID
			for _, rule := range res.Rules {
				ruleCount += 1
				rule.ID = ruleCount
				rule.ResourceID = &res.ID
			}
		}
	}

	out, _ := yaml.Marshal(tpl)
	os.WriteFile("config/promql_tpl.yaml", out, fs.FileMode(os.O_WRONLY))
}
