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
	"database/sql/driver"
	"encoding/json"
)

func (g *MonitorGraphs) Scan(src interface{}) error {
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, g)
	}
	return nil
}

// 注意这里不是指针，下同
func (g MonitorGraphs) Value() (driver.Value, error) {
	return json.Marshal(g)
}

func (g MonitorGraphs) GormDataType() string {
	return "json"
}

type MonitorGraphs []MetricGraph
type Target struct {
	TargetName      string           `json:"targetName"`
	PromqlGenerator *PromqlGenerator `json:"promqlGenerator"`
	Expr            string           `json:"expr"`
}

type MetricGraph struct {
	Name    string   `json:"name"`
	Targets []Target `json:"targets"`
	Unit    string   `json:"unit"`
}

func (graphs MonitorGraphs) IsUsingTpl(scope, resource, rule string) bool {
	for _, graph := range graphs {
		for _, t := range graph.Targets {
			if t.PromqlGenerator != nil &&
				t.PromqlGenerator.Scope == scope &&
				t.PromqlGenerator.Resource == resource &&
				t.PromqlGenerator.Rule == rule {
				return true
			}
		}
	}
	return false
}
