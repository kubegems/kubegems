package msgbus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/msgbus"
)

func NewMessageBusClient(database *database.Database, options *msgbus.Options) *MsgBusClient {
	return &MsgBusClient{
		Database: database,
		options:  options,
		httpclient: &http.Client{
			Timeout: 5 * time.Second,
		},
		messageBusServer: options.Addr + "/v2/msgbus/send",
	}
}

type MsgBusClient struct {
	c                *gin.Context
	options          *msgbus.Options
	messageBusServer string
	httpclient       *http.Client
	Database         *database.Database
	messageType      msgbus.MessageType
	resourceType     msgbus.ResourceType
	actionType       msgbus.EventKind
	resourceID       uint
	content          string
	toUsers          []uint
	affectedUsers    []uint
}

func (cli *MsgBusClient) GinContext(c *gin.Context) *MsgBusClient {
	cli.c = c.Copy() // goroutine中使用copy
	return cli
}

func (cli *MsgBusClient) MessageType(messageType msgbus.MessageType) *MsgBusClient {
	cli.messageType = messageType
	return cli
}

func (cli *MsgBusClient) ResourceType(resourceType msgbus.ResourceType) *MsgBusClient {
	cli.resourceType = resourceType
	return cli
}

func (cli *MsgBusClient) ActionType(actionType msgbus.EventKind) *MsgBusClient {
	cli.actionType = actionType
	return cli
}

func (cli *MsgBusClient) ResourceID(resourceID uint) *MsgBusClient {
	cli.resourceID = resourceID
	return cli
}

func (cli *MsgBusClient) Content(content string) *MsgBusClient {
	cli.content = content
	return cli
}

func (cli *MsgBusClient) SetUsersToSend(idSlices ...[]uint) *MsgBusClient {
	idSet := make(map[uint]struct{}) // 去重
	for _, ids := range idSlices {
		for _, id := range ids {
			if _, ok := idSet[id]; !ok {
				cli.toUsers = append(cli.toUsers, id)
				idSet[id] = struct{}{}
			}
		}
	}
	return cli
}

func (cli *MsgBusClient) AffectedUsers(idSlices ...[]uint) *MsgBusClient {
	idSet := make(map[uint]struct{}) // 去重
	for _, ids := range idSlices {
		for _, id := range ids {
			if _, ok := idSet[id]; !ok {
				cli.affectedUsers = append(cli.affectedUsers, id)
				idSet[id] = struct{}{}
			}
		}
	}
	return cli
}

func (cli *MsgBusClient) Send() {
	go func() {
		msg := msgbus.NotifyMessage{
			MessageType: cli.messageType,
			EventKind:   cli.actionType,
		}
		content := msgbus.MessageContent{
			ResourceType:  cli.resourceType,
			ResouceID:     cli.resourceID,
			AffectedUsers: cli.affectedUsers,
			CreatedAt:     time.Now(),
			To:            cli.toUsers,
		}
		from, ok := cli.c.Get("current_user")
		if ok {
			content.From = from.(*models.User).Username
			content.Detail = fmt.Sprintf("用户%s%s", content.From, cli.content)
		}
		msg.Content = content

		o, _ := json.Marshal(msgbus.MessageTarget{
			Message: msg,
			Users:   cli.toUsers,
		})
		body := bytes.NewBuffer(o)

		// all send to msgbus
		req, err := http.NewRequest(http.MethodPost, cli.messageBusServer, body)
		if err != nil {
			log.Error(err, "new request")
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", cli.c.GetHeader("Authorization"))
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
			contentJson, _ := json.Marshal(content)
			dbmsg := models.Message{
				MessageType: string(msg.MessageType),
				Title:       content.Detail,
				CreatedAt:   now,
				Content:     contentJson,
			}
			if err := cli.Database.DB().Save(&dbmsg).Error; err != nil {
				log.Error(err, "save db message")
				return
			}

			usermsgs := make([]models.UserMessageStatus, len(cli.toUsers))
			for i := range cli.toUsers {
				usermsgs[i].UserID = cli.toUsers[i]
				usermsgs[i].MessageID = &dbmsg.ID
				usermsgs[i].IsRead = false
			}
			if err := cli.Database.DB().Create(&usermsgs).Error; err != nil {
				log.Error(err, "save user message")
				return
			}
		}
	}()
}
