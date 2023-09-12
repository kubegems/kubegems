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

package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"kubegems.io/kubegems/pkg/msgbus/switcher"
	"kubegems.io/kubegems/pkg/service/aaa"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/library/rest/response"
)

type MessageHandler struct {
	*aaa.UserInfoHandler
	Switcher *switcher.MessageSwitcher
}

func (m *MessageHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/msgbus/notify", m.MessageCenter)
	rg.POST("/msgbus/send", m.SendMessages)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// @Tags			MSGBUS
// @Summary		消息中心(websocket)
// @Description	消息中心(websocket)
// @Accept			json
// @Produce		json
// @Success		200	{object}	object	"stream"
// @Router			/realtime/v2/msgbus/notify [get]
// @Security		JWT
func (m *MessageHandler) MessageCenter(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	var user *switcher.NotifyUser
	dbUser, exist := m.GetContextUser(c)
	if exist {
		user = switcher.NewNotifyUser(conn, dbUser.GetUsername(), dbUser.GetID())
	} else {
		user = switcher.NewNotifyUser(conn, "none", 0)
	}
	m.HandleMessage(c.Request.Context(), m.Switcher, user)
}

func (m *MessageHandler) HandleMessage(ctx context.Context, ms *switcher.MessageSwitcher, nu *switcher.NotifyUser) {
	ms.RegistUser(nu)
	defer ms.DeRegistUser(nu)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg := &msgbus.ControlMessage{}
			if err := nu.Read(msg); err != nil {
				return
			}
			switch msg.MessageType {
			case msgbus.Changed:
				nu.SetCurrentWatch(msg.Content)
			}
		}
	}
}

// @Tags			MSGBUS
// @Summary		发送消息
// @Description	发送消息
// @Accept			json
// @Produce		json
// @Param			param	body		msgbus.MessageTarget	true	"消息"
// @Success		200		{object}	object					"stream"
// @Router			/realtime/v2/msgbus/send [post]
// @Security		JWT
func (m *MessageHandler) SendMessages(c *gin.Context) {
	var msgTarget msgbus.MessageTarget
	if err := c.Bind(&msgTarget); err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	for _, uid := range msgTarget.Users {
		m.Switcher.SendMessageToUser(&msgTarget.Message, uid)
	}
	handlers.OK(c, "ok")
}
