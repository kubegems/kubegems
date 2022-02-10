package models

import (
	"strconv"
	"time"

	"github.com/kubegems/gems/pkg/utils"
	"github.com/kubegems/gems/pkg/utils/msgbus"
	"gorm.io/datatypes"
)

type AlertInfo struct {
	Fingerprint string `gorm:"type:varchar(50);primaryKey"` // 指纹作为主键
	Name        string `gorm:"type:varchar(50);"`
	Namespace   string `gorm:"type:varchar(50);"`
	ClusterName string `gorm:"type:varchar(50);"`
	Labels      datatypes.JSON
	LabelMap    map[string]string `gorm:"-" json:"-"`

	SilenceStartsAt  *time.Time
	SilenceUpdatedAt *time.Time
	SilenceEndsAt    *time.Time
	SilenceCreator   string `gorm:"type:varchar(50);"`
	Summary          string `gorm:"-"` // 黑名单概要
}

type AlertMessage struct {
	ID uint

	// 级联删除
	Fingerprint string     `gorm:"type:varchar(50);"`
	AlertInfo   *AlertInfo `gorm:"foreignKey:Fingerprint;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Value     string
	Message   string
	StartsAt  *time.Time `gorm:"index"` // 告警开始时间
	EndsAt    *time.Time // 告警结束时间
	CreatedAt *time.Time `gorm:"index"` // 本次告警产生时间
	Status    string     // firing or resolved
}

func (a *AlertMessage) ToNormalMessage() Message {
	return Message{
		ID:          a.ID,
		MessageType: string(msgbus.Alert),
		Title:       a.Message,
		CreatedAt:   *a.CreatedAt,
	}
}

func (a *AlertMessage) ColumnSlice() []string {
	return []string{"id", "fingerprint", "value", "message", "starts_at", "ends_at", "created_at", "status", "labels"}
}

func (a *AlertMessage) ValueSlice() []string {
	return []string{
		strconv.Itoa(int(a.ID)),
		a.Fingerprint,
		a.Value,
		a.Message,
		utils.FormatMysqlDumpTime(a.StartsAt),
		utils.FormatMysqlDumpTime(a.EndsAt),
		utils.FormatMysqlDumpTime(a.CreatedAt),
		a.Status,
		func() string {
			if a.AlertInfo != nil {
				return a.AlertInfo.Labels.String()
			}
			return ""
		}(),
	}
}
