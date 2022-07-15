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

package switcher

import (
	"encoding/json"
	"fmt"
	"time"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

type WebhookAlert struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int64             `json:"truncatedAlerts"`
	DataBase          *database.Database
}

func (w *WebhookAlert) fingerprintMap() map[string][]Alert {
	ret := map[string][]Alert{}
	for _, v := range w.Alerts {
		alerts, ok := ret[v.Fingerprint]
		if ok {
			alerts = append(alerts, v)
		} else {
			alerts = []Alert{v}
		}
		ret[v.Fingerprint] = alerts
	}

	return ret
}

func (ms *MessageSwitcher) saveFingerprintMapToDB(fingerprintMap map[string][]Alert) []models.AlertMessage {
	now := time.Now()
	alertMessages := []models.AlertMessage{}
	alertInfos := []models.AlertInfo{}

	clusterNS2EnvMap, err := ms.DataBase.ClusterNS2EnvMap()
	if err != nil {
		log.Error(err, "get ClusterNS2EnvMap")
	}
	for fingerprint, alerts := range fingerprintMap {
		labelbyts, _ := json.Marshal(alerts[0].Labels)
		envinfo := clusterNS2EnvMap[fmt.Sprintf("%s/%s",
			alerts[0].Labels[prometheus.AlertClusterKey],
			alerts[0].Labels[prometheus.AlertNamespaceLabel])]

		tmpAlertInfo := models.AlertInfo{
			Fingerprint:     fingerprint,
			Name:            alerts[0].Labels[prometheus.AlertNameLabel], // 铁定有元素的，不会越界
			Namespace:       alerts[0].Labels[prometheus.AlertNamespaceLabel],
			ClusterName:     alerts[0].Labels[prometheus.AlertClusterKey],
			TenantName:      envinfo.TenantName,
			ProjectName:     envinfo.ProjectName,
			EnvironmentName: envinfo.EnvironmentName,
			Labels:          labelbyts,
		}
		alertInfos = append(alertInfos, tmpAlertInfo)

		for _, alert := range alerts {
			alertMessages = append(alertMessages, models.AlertMessage{
				Fingerprint: fingerprint,
				Value:       alert.Annotations["value"],
				Message:     alert.Annotations["message"],
				StartsAt:    utils.TimeZeroToNull(alert.StartsAt),
				EndsAt:      utils.TimeZeroToNull(alert.EndsAt),
				CreatedAt:   &now,
				Status:      alert.Status,
				AlertInfo:   &tmpAlertInfo,
			})
		}
	}
	if err := ms.DataBase.DB().Save(&alertInfos).Error; err != nil {
		log.Error(err, "save alert info")
		return nil
	}
	if err := ms.DataBase.DB().Save(&alertMessages).Error; err != nil {
		log.Error(err, "save alert message")
		return nil
	}
	return alertMessages
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     *time.Time        `json:"startsAt"`
	EndsAt       *time.Time        `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

func (a *Alert) AlertName() string {
	return a.Labels["alertname"]
}

func (a *Alert) Detail() string {
	return fmt.Sprintf("%s: %s", a.AlertName(), a.Annotations["message"])
}

type ResID struct {
	ClusterID       uint
	EnvironmentID   uint
	EnvironmentName string
	ProjectID       uint
	ProjectName     string
	TenantID        uint
	TenantName      string
}

// map 效率更高
func (w *WebhookAlert) GetAlertUsers(pos database.AlertPosition) map[uint]struct{} {
	tmp := []uint{}
	switch w.CommonLabels["gems_alert_scope"] {
	case prometheus.ScopeSystemAdmin:
		tmp = append(tmp, w.DataBase.SystemAdmins()...) // 系统管理员
	case prometheus.ScopeSystemUser:
		tmp = append(tmp, w.DataBase.SystemUsers()...) // 系统所有用户
	default: // normal and null
		tmp = append(tmp, w.DataBase.EnvUsers(pos.EnvironmentID)...)  // 环境用户
		tmp = append(tmp, w.DataBase.ProjectAdmins(pos.ProjectID)...) // 项目管理员
		tmp = append(tmp, w.DataBase.TenantAdmins(pos.TenantID)...)   // 租户管理员
	}
	ret := make(map[uint]struct{})
	for _, v := range tmp {
		ret[v] = struct{}{}
	}
	return ret
}
