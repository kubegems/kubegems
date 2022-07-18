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

package msgbus

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/set"
)

func NewMessageBusClient(database *database.Database, options *msgbus.Options) *MsgBusClient {
	return &MsgBusClient{
		Database: database,
		httpclient: &http.Client{
			Timeout: 5 * time.Second,
		},
		messageBusServer: options.Addr + "/v2/msgbus/send",
	}
}

type MsgBusClient struct {
	messageBusServer string
	httpclient       *http.Client
	Database         *database.Database
}

type MsgRequest struct {
	msgbus.MessageType
	msgbus.ResourceType
	msgbus.EventKind

	Username      string
	Authorization string
	ResourceID    uint
	Detail        string

	ToUsers       *set.Set[uint]
	AffectedUsers *set.Set[uint]
}

func (cli *MsgBusClient) Send(msgreq *MsgRequest) {
	msg := msgbus.NotifyMessage{
		MessageType: msgreq.MessageType,
		EventKind:   msgreq.EventKind,
		Content: msgbus.MessageContent{
			ResourceType: msgreq.ResourceType,
			ResouceID:    msgreq.ResourceID,
			From:         msgreq.Username,
			Detail:       msgreq.Detail,
			CreatedAt:    time.Now(),

			To:            msgreq.ToUsers.Slice(),
			AffectedUsers: msgreq.AffectedUsers.Slice(),
		},
	}

	o, _ := json.Marshal(msgbus.MessageTarget{
		Message: msg,
		Users:   msgreq.ToUsers.Slice(),
	})
	body := bytes.NewBuffer(o)

	// all send to msgbus
	req, err := http.NewRequest(http.MethodPost, cli.messageBusServer, body)
	if err != nil {
		log.Error(err, "new request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", msgreq.Authorization)
	resp, err := cli.httpclient.Do(req)
	if err != nil {
		log.Error(err, "send msgbus")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		tmp, _ := io.ReadAll(resp.Body)
		err := errors.New(string(tmp))
		log.Error(err, "send msgbus")
		return
	}

	// message save to DB
	now := time.Now()
	if msg.MessageType == msgbus.Message || msg.MessageType == msgbus.Approve {
		contentJson, _ := json.Marshal(msg.Content)
		dbmsg := models.Message{
			MessageType: string(msg.MessageType),
			Title:       msgreq.Detail,
			CreatedAt:   now,
			Content:     contentJson,
		}
		if err := cli.Database.DB().Save(&dbmsg).Error; err != nil {
			log.Error(err, "save db message")
			return
		}

		tousers := msgreq.ToUsers.Slice()
		usermsgs := make([]models.UserMessageStatus, len(tousers))
		for i := range tousers {
			usermsgs[i].UserID = tousers[i]
			usermsgs[i].MessageID = &dbmsg.ID
			usermsgs[i].IsRead = false
		}
		if err := cli.Database.DB().Create(&usermsgs).Error; err != nil {
			log.Error(err, "save user message")
			return
		}
	}
}
