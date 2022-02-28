package models

import (
	"fmt"
	"os"
	"path"
	"time"

	"gorm.io/gorm"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/loki"
)

// LogQueryHistory
type LogQueryHistory struct {
	ID         uint     `gorm:"primarykey"`
	Cluster    *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ClusterID  uint
	LabelJSON  string    `gorm:"type:varchar(1024)"`
	FilterJSON string    `gorm:"type:varchar(1024)"`
	LogQL      string    `gorm:"type:varchar(1024)"`
	CreateAt   time.Time `sql:"DEFAULT:'current_timestamp'"`
	Creator    *User
	CreatorID  uint
}

type LogQueryHistoryWithCount struct {
	ID         uint
	Ids        string
	Cluster    *Cluster
	ClusterID  uint
	LabelJSON  string
	FilterJSON string
	LogQL      string
	CreateAt   time.Time
	Creator    *User
	CreatorID  uint
	Total      string
}

type LogQuerySnapshot struct {
	ID           uint     `gorm:"primarykey"`
	Cluster      *Cluster `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:CASCADE;"`
	ClusterID    uint
	SnapshotName string `gorm:"type:varchar(128)"`
	SourceFile   string `gorm:"type:varchar(128)"`
	// line count
	SnapshotCount int
	DownloadURL   string `gorm:"type:varchar(512)"`
	StartTime     time.Time
	EndTime       time.Time
	CreateAt      time.Time `sql:"DEFAULT:'current_timestamp'"`
	Creator       *User
	CreatorID     uint
}

func (snapshot *LogQuerySnapshot) BeforeCreate(tx *gorm.DB) error {
	var (
		lineCount int64
		err       error
	)
	lokiExportDir := "lokiExport"

	lokiSnapshotDir := path.Join(lokiExportDir, "snapshot", time.Now().UTC().Format("20060102"))
	err = utils.EnsurePathExists(lokiSnapshotDir)
	if err != nil {
		return err
	}
	sourceFile := path.Join(lokiExportDir, snapshot.SourceFile)
	if loki.FileExists(sourceFile) {
		targetFile := path.Join(lokiSnapshotDir, snapshot.SnapshotName)
		if !loki.FileExists(targetFile) {
			lineCount, err = utils.CopyFileByLine(targetFile, sourceFile)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("名字为 %s 的日志快照 已经存在，请换个名字保存", snapshot.SnapshotName)
		}
		snapshot.SnapshotCount = int(lineCount)
		snapshot.DownloadURL = path.Join("/", lokiSnapshotDir, snapshot.SnapshotName)
		return nil
	}

	return nil
}

func (snapshot *LogQuerySnapshot) BeforeDelete(tx *gorm.DB) error {
	if snapshot.DownloadURL != "" && loki.FileExists(snapshot.DownloadURL) {
		err := os.Remove(snapshot.DownloadURL)
		if err != nil {
			return err
		}
	}
	return nil
}
