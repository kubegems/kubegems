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

package orm

import "time"

// +gen type:object pkcolume:id pkfield:ID preloads:Cluster,Creator
type LogQueryHistory struct {
	ID         uint     `gorm:"primarykey"`
	Cluster    *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ClusterID  uint
	LabelJSON  string     `gorm:"type:varchar(1024)"`
	FilterJSON string     `gorm:"type:varchar(1024)"`
	LogQL      string     `gorm:"type:varchar(1024)"`
	CreateAt   *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Creator    *User
	CreatorID  uint
}

// +gen type:object pkcolume:id pkfield:ID preloads:Cluster,Creator
type LogQuerySnapshot struct {
	ID            uint     `gorm:"primarykey"`
	Name          string   `gorm:"type:varchar(128)"`
	Cluster       *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ClusterID     uint
	SourceFile    string `gorm:"type:varchar(128)"`
	SnapshotCount int    // file line count
	DownloadURL   string `gorm:"type:varchar(512)"`
	StartTime     *time.Time
	EndTime       *time.Time
	CreateAt      *time.Time `sql:"DEFAULT:'current_timestamp'"`
	Creator       *User
	CreatorID     uint
}
