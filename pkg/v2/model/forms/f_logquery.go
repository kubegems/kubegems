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

package forms

import "time"

// +genform object:LogQueryHistory
type LogQueryHistoryCommon struct {
	BaseForm
	ID         uint           `json:"id,omitempty"`
	Cluster    *ClusterCommon `json:"cluster,omitempty"`
	ClusterID  uint           `json:"clusterID,omitempty"`
	LabelJSON  string         `json:"labelJSON,omitempty"`
	FilterJSON string         `json:"filterJSON,omitempty"`
	LogQL      string         `json:"logQL,omitempty"`
	CreateAt   *time.Time     `json:"createAt,omitempty"`
	Creator    *UserCommon    `json:"creator,omitempty"`
	CreatorID  uint           `json:"creatorID,omitempty"`
}

// +genform object:LogQuerySnapshot
type LogQuerySnapshotCommon struct {
	BaseForm
	ID            uint           `json:"id,omitempty"`
	Cluster       *ClusterCommon `json:"cluster,omitempty"`
	ClusterID     uint           `json:"clusterID,omitempty"`
	Name          string         `json:"name,omitempty"`
	SourceFile    string         `json:"sourceFile,omitempty"`
	SnapshotCount int            `json:"snapshotCount,omitempty"`
	DownloadURL   string         `json:"downloadURL,omitempty"`
	StartTime     *time.Time     `json:"startTime,omitempty"`
	EndTime       *time.Time     `json:"endTime,omitempty"`
	CreateAt      *time.Time     `json:"createAt,omitempty"`
	Creator       *UserCommon    `json:"creator,omitempty"`
	CreatorID     uint           `json:"creatorID,omitempty"`
}
