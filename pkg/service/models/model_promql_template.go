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

package models

import (
	"fmt"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
	"kubegems.io/kubegems/pkg/utils/gormdatatypes"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
	"kubegems.io/kubegems/pkg/utils/set"
)

// PromqlTplScope
type PromqlTplScope struct {
	ID         uint   `gorm:"primarykey" json:"id"`
	Name       string `gorm:"type:varchar(50);uniqueIndex" binding:"required" json:"name"`
	ShowName   string `gorm:"type:varchar(50)" json:"showName"`
	Namespaced bool   `json:"namespaced"`

	Resources []*PromqlTplResource `json:"resources" gorm:"foreignKey:ScopeID"`

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// PromqlTplScope
type PromqlTplResource struct {
	ID       uint   `gorm:"primarykey" json:"id"`
	Name     string `gorm:"type:varchar(50);uniqueIndex" binding:"required" json:"name"`
	ShowName string `gorm:"type:varchar(50)" json:"showName"`

	ScopeID *uint           `json:"scopeID"`
	Scope   *PromqlTplScope `gorm:"foreignKey:ScopeID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"scope,omitempty"`

	Rules []*PromqlTplRule `json:"rules" gorm:"foreignKey:ResourceID"`

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

// PromqlTplScope
type PromqlTplRule struct {
	ID          uint                    `gorm:"primarykey" json:"id"`
	Name        string                  `gorm:"type:varchar(50)" binding:"required" json:"name"`
	ShowName    string                  `gorm:"type:varchar(50)" json:"showName"`
	Description string                  `json:"description"`
	Expr        string                  `json:"expr"` // promql expr
	Unit        string                  `gorm:"type:varchar(50)" json:"unit"`
	Labels      gormdatatypes.JSONSlice `json:"labels"`

	ResourceID *uint              `json:"resourceID"`
	Resource   *PromqlTplResource `gorm:"foreignKey:ResourceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"resource,omitempty"`

	TenantID *uint   `json:"tenantID"` // 若为null，则表示系统预置
	Tenant   *Tenant `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;" json:"tenant,omitempty"`

	CreatedAt *time.Time `json:"-"`
	UpdatedAt *time.Time `json:"-"`
}

func CheckGraphs(graphs []prometheus.MetricGraph, namespace string, tplGetter templates.TplGetter) error {
	// 逐个校验graph
	for i, graph := range graphs {
		if graph.Name == "" {
			return fmt.Errorf("图表名不能为空")
		}
		if graph.Unit != "" {
			if _, err := prometheus.ParseUnit(graph.Unit); err != nil {
				return err
			}
		}
		for j, target := range graph.Targets {
			targetSet := set.NewSet[string]()
			if targetSet.Has(target.TargetName) {
				return fmt.Errorf("duplicated target name: %s", target.TargetName)
			}
			targetSet.Append(target.TargetName)

			if target.PromqlGenerator.Notpl() {
				if target.Expr == "" {
					return fmt.Errorf("模板与原生promql不能同时为空")
				}
				query, err := promql.New(target.Expr)
				if err != nil {
					return err
				}
				if namespace != "" {
					// 强制添加namespace selector
					graphs[i].Targets[j].Expr = query.AddLabelMatchers(&labels.Matcher{
						Type:  labels.MatchEqual,
						Name:  "namespace",
						Value: namespace,
					}).String()
				}

			} else {
				if err := target.PromqlGenerator.SetTpl(tplGetter); err != nil {
					return err
				}
				if !target.PromqlGenerator.Tpl.Namespaced {
					return fmt.Errorf("图表: %s-%s 错误！不能查询集群范围资源", graph.Name, target.TargetName)
				}
				graphs[i].Unit = target.PromqlGenerator.Unit
			}
		}
	}
	return nil
}
