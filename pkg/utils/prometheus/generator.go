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
	"fmt"
	"regexp"

	"github.com/prometheus/prometheus/pkg/labels"
	"kubegems.io/kubegems/pkg/utils/prometheus/promql"
	"kubegems.io/kubegems/pkg/utils/prometheus/templates"
	"kubegems.io/kubegems/pkg/utils/slice"
)

type PromqlGenerator struct {
	Scope      string            `json:"scope"`                // scope
	Resource   string            `json:"resource"`             // 告警资源, eg. node、pod
	Rule       string            `json:"rule"`                 // 告警规则名, eg. cpuUsage、memoryUsagePercent
	Unit       string            `json:"unit"`                 // 单位
	LabelPairs map[string]string `json:"labelpairs,omitempty"` // 标签键值对

	UnitValue UnitValue `json:"-"`

	Tpl *templates.PromqlTpl `json:"-"`
}

var reg = regexp.MustCompile(`^\w+$`)

func IsValidPromqlTplName(scope, resource, rule string) error {
	if !reg.MatchString(scope) {
		return fmt.Errorf("scope not valid, must match regex: %s", reg.String())
	}
	if !reg.MatchString(resource) {
		return fmt.Errorf("resource not valid, must match regex: %s", reg.String())
	}
	if !reg.MatchString(rule) {
		return fmt.Errorf("rule not valid, must match regex: %s", reg.String())
	}
	return nil
}

func (g *PromqlGenerator) Notpl() bool {
	return g == nil || g.Resource == ""
}

func (g *PromqlGenerator) TplString() string {
	return fmt.Sprintf("%s.%s.%s", g.Scope, g.Resource, g.Rule)
}

func (g *PromqlGenerator) SetTpl(f templates.TplGetter) error {
	if err := IsValidPromqlTplName(g.Scope, g.Resource, g.Rule); err != nil {
		return err
	}
	tpl, err := f(g.Scope, g.Resource, g.Rule)
	if err != nil {
		return err
	}
	for label := range g.LabelPairs {
		if !slice.ContainStr(tpl.Labels, label) {
			return fmt.Errorf("label: %s not in tpl: %v", label, tpl.String())
		}
	}
	if g.Unit == "" {
		g.Unit = tpl.Unit
	} else if g.Unit != tpl.Unit {
		return fmt.Errorf("unit: %s not euqal with unit in template: %s", g.Unit, tpl.Unit)
	}

	unitValue, err := ParseUnit(g.Unit)
	if err != nil {
		return err
	}

	g.UnitValue = unitValue
	g.Tpl = tpl
	return nil
}

func (g *PromqlGenerator) ToPromql(namespace string) (string, error) {
	query, err := promql.New(g.Tpl.Expr)
	if err != nil {
		return "", err
	}
	ls := map[string]string{}
	for k, v := range g.LabelPairs {
		ls[k] = v
	}
	if namespace != GlobalAlertNamespace && namespace != "" {
		// force add namespace label
		ls[PromqlNamespaceKey] = namespace
	}

	for label, value := range ls {
		query.AddLabelMatchers(&labels.Matcher{
			Type:  labels.MatchRegexp,
			Name:  label,
			Value: value,
		})
	}

	return query.String(), nil
}
