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

type NodeType string

const (
	Matcher      NodeType = "matcher"
	Filter       NodeType = "filter"
	Output       NodeType = "output"
	GlobalOutput NodeType = "globaloutput"
)

type MetricTable struct {
	Rows []MetricRow `json:"rows"`
}

type MetricRow struct {
	AppName      string  `json:"appName"`
	RealTimeRate float64 `json:"realTimeRate"`
	AvgOfHour    float64 `json:"avgOfHour"`
	AvgOfDay     float64 `json:"avgOfDay"`
}
type Graph struct {
	GraphType string   `json:"graphType"`
	Elements  Elements `json:"elements"`
}

type Elements struct {
	Nodes []*NodeWrapper `json:"nodes"`
	Edges []*EdgeWrapper `json:"edges"`
}

type NodeWrapper struct {
	Data *NodeData `json:"data"`
}

type EdgeWrapper struct {
	Data *EdgeData `json:"data"`
}

type NodeData struct {
	// Cytoscape Fields
	ID     string `json:"id"`               // unique internal node ID (n0, n1...)
	Parent string `json:"parent,omitempty"` // Compound Node parent ID
	// NOTE: add new custom field
	NodeType string `json:"nodeType"`
}

type EdgeData struct {
	// Cytoscape Fields
	ID     string `json:"id"`     // unique internal edge ID (e0, e1...)
	Source string `json:"source"` // parent node ID
	Target string `json:"target"` // child node ID
	//NOTE: add new custom field
}
