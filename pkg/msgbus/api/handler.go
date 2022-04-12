package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"kubegems.io/pkg/msgbus/switcher"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/handlers"
	"kubegems.io/pkg/utils/msgbus"
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

// @Tags         MSGBUS
// @Summary      消息中心(websocket)
// @Description  消息中心(websocket)
// @Accept       json
// @Produce      json
// @Success      200  {object}  object  "stream"
// @Router       /realtime/v2/msgbus/notify [get]
// @Security     JWT
func (m *MessageHandler) MessageCenter(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{"Message": err.Error()})
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

// @Tags         MSGBUS
// @Summary      发送消息
// @Description  发送消息
// @Accept       json
// @Produce      json
// @Param        param  body      msgbus.MessageTarget  true  "消息"
// @Success      200    {object}  object                "stream"
// @Router       /realtime/v2/msgbus/send [post]
// @Security     JWT
func (m *MessageHandler) SendMessages(c *gin.Context) {
	var msgTarget msgbus.MessageTarget
	if err := c.Bind(&msgTarget); err != nil {
		c.AbortWithStatusJSON(400, gin.H{"Message": err.Error()})
		return
	}
	for _, uid := range msgTarget.Users {
		m.Switcher.SendMessageToUser(&msgTarget.Message, uid)
	}
	handlers.OK(c, "ok")
}
