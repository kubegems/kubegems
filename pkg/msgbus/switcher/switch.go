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
	"context"
	"encoding/json"
	"time"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/prometheus"
)

func NewMessageSwitch(_ context.Context, db *database.Database) *MessageSwitcher {
	messageSwitcher := &MessageSwitcher{
		Users:    []*NotifyUser{},
		DataBase: db,
	}
	return messageSwitcher
}

type MessageSwitcher struct {
	DataBase *database.Database
	Users    []*NotifyUser
}

func (ms *MessageSwitcher) RegistUser(user *NotifyUser) {
	ms.Users = append(ms.Users, user)
}

func (ms *MessageSwitcher) DeRegistUser(user *NotifyUser) {
	index := 0
	for idx := range ms.Users {
		if ms.Users[idx].SessionID != user.SessionID {
			ms.Users[index] = ms.Users[idx]
			index++
		} else {
			ms.Users[idx].CloseConn()
		}
	}
	ms.Users = ms.Users[:index]
}

func (ms *MessageSwitcher) DispatchMessage(msg *msgbus.NotifyMessage) {
	switch msg.MessageType {
	case msgbus.Alert:
		webhookAlert := WebhookAlert{DataBase: ms.DataBase}
		b, ok := msg.Content.(string)
		if !ok {
			log.Errorf("content type is not string: %s", msg.Content)
			return
		}
		if err := json.Unmarshal([]byte(b), &webhookAlert); err != nil {
			log.Error(err, "json unmarshal error")
			return
		}

		// 存告警消息表
		fingerprintMap := webhookAlert.fingerprintMap()
		dbalertMsgs := ms.saveFingerprintMapToDB(fingerprintMap)

		// 发消息并存用户消息表
		pos, _ := ms.DataBase.GetAlertPosition(
			webhookAlert.CommonLabels[prometheus.AlertClusterKey],
			webhookAlert.CommonLabels[prometheus.AlertNamespaceLabel],
			webhookAlert.CommonLabels[prometheus.AlertNameLabel],
			webhookAlert.CommonLabels[prometheus.AlertScopeLabel],
			webhookAlert.CommonLabels[prometheus.AlertFromLabel],
		)
		toUsers := webhookAlert.GetAlertUsers(pos)
		now := time.Now()
		dbUserMsgs := []models.UserMessageStatus{}
		// save之后有了ID，才能做关联
		for i := range dbalertMsgs {
			// 发送消息
			for _, u := range ms.Users {
				if _, ok := toUsers[u.UserID]; ok {
					ms.Send(u, &msgbus.NotifyMessage{
						MessageType: msgbus.Alert,
						Content: msgbus.MessageContent{
							CreatedAt: now,
							From:      dbalertMsgs[i].AlertInfo.Name,
							Detail:    dbalertMsgs[i].Message,
						},
					})
				}
			}

			// 存用户消息表
			usermsgs := make([]models.UserMessageStatus, len(toUsers))
			index := 0
			for id := range toUsers {
				usermsgs[index].UserID = id
				usermsgs[index].AlertMessageID = &dbalertMsgs[i].ID
				usermsgs[index].IsRead = false
				index++
			}
			dbUserMsgs = append(dbUserMsgs, usermsgs...)
		}

		if err := ms.DataBase.DB().Save(&dbUserMsgs).Error; err != nil {
			log.Error(err, "save user message status")
			return
		}

		log.Infof("receive cluster [%s] namespace [%s] alert [%s], alert count %d, user message count: %d",
			webhookAlert.CommonLabels[prometheus.AlertClusterKey],
			webhookAlert.CommonLabels[prometheus.AlertNamespaceLabel],
			webhookAlert.CommonLabels[prometheus.AlertNameLabel],
			len(webhookAlert.Alerts),
			len(dbUserMsgs),
		)
	case msgbus.Changed:
		for _, u := range ms.Users {
			if u.IsWatchObject(msg) {
				_ = ms.Send(u, msg)
			}
		}
	}
}

func (ms *MessageSwitcher) SendMessageToUser(msg *msgbus.NotifyMessage, userid uint) {
	for _, u := range ms.Users {
		if u.UserID == userid {
			_ = ms.Send(u, msg)
		}
	}
}

func (ms *MessageSwitcher) Send(user *NotifyUser, msg *msgbus.NotifyMessage) error {
	err := user.Write(msg)
	if err != nil {
		ms.DeRegistUser(user)
		return err
	}
	return nil
}
