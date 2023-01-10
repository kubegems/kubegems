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
	"kubegems.io/kubegems/pkg/utils/set"
)

func (ms *MessageSwitcher) saveFingerprintMapToDB(fingerprintMap map[string][]prometheus.Alert) []models.AlertMessage {
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
func GetAlertUsers(pos database.AlertPosition, database *database.Database) *set.Set[uint] {
	ret := set.NewSet[uint]()
	if pos.Namespace == prometheus.GlobalAlertNamespace {
		ret.Append(database.SystemAdmins()...) // 系统管理员
	} else {
		// 环境用户、项目管理员、租户管理员
		ret.Append(database.EnvUsers(pos.EnvironmentID)...).
			Append(database.ProjectAdmins(pos.ProjectID)...).
			Append(database.TenantAdmins(pos.TenantID)...)
	}
	return ret
}
