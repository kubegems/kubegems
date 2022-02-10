package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"kubegems.io/pkg/log"
)

var Upgrader = websocket.Upgrader{
	// 允许所有CORS跨域请求
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// websocket消息
type WsMessage struct {
	MessageType int
	Data        []byte
}

type WsConnection struct {
	conn    *websocket.Conn
	inChan  chan *WsMessage
	outChan chan *WsMessage
	cancel  context.CancelFunc
	lock    sync.RWMutex
	stoped  bool
	OnClose func()
}

func (wsConn *WsConnection) wsReadLoop(ctx context.Context) {
	var (
		msgType int
		data    []byte
		err     error
	)
	for {
		if msgType, data, err = wsConn.conn.ReadMessage(); err != nil {
			log.Errorf("failed to read websocket msg %v", err)
			wsConn.WsClose()
			return
		}
		xmsg := xtermMessage{}
		e := json.Unmarshal(data, &xmsg)
		if e == nil {
			if xmsg.MsgType == "close" {
				closeMsg, _ := json.Marshal(xtermMessage{
					MsgType: "input",
					Input:   "exit\r",
				})
				wsConn.inChan <- &WsMessage{MessageType: msgType, Data: closeMsg}
			}
		}
		wsConn.inChan <- &WsMessage{MessageType: msgType, Data: data}
		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (wsConn *WsConnection) wsWriteLoop(ctx context.Context) {
	var (
		msg *WsMessage
		err error
	)

	for {
		select {
		case msg = <-wsConn.outChan:
			if msg != nil {
				if err = wsConn.conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
					log.Errorf("failed to write websocket msg %v", err)
					wsConn.WsClose()
				}
			}
		case <-ctx.Done():
			log.Infof("stop write loop")
			return
		}
	}
}

func (wsConn *WsConnection) WsWrite(messageType int, data []byte) (err error) {
	wsConn.lock.RLock()
	defer wsConn.lock.RUnlock()
	if wsConn.stoped {
		err = errors.New("can't write on closed channel")
		return
	}
	wsConn.outChan <- &WsMessage{messageType, data}
	return
}

func (wsConn *WsConnection) WsRead() (msg *WsMessage, err error) {
	wsConn.lock.RLock()
	defer wsConn.lock.RUnlock()
	if wsConn.stoped {
		err = errors.New("can't read on closed channel")
		return
	}
	msg = <-wsConn.inChan
	return
}

func (wsConn *WsConnection) WsClose() {
	if wsConn.OnClose != nil {
		wsConn.OnClose()
	}
	wsConn.lock.Lock()
	defer wsConn.lock.Unlock()
	if wsConn.stoped {
		return
	}
	wsConn.stoped = true
	wsConn.cancel()
	wsConn.conn.Close()
	close(wsConn.inChan)
	close(wsConn.outChan)
}

func InitWebsocket(resp http.ResponseWriter, req *http.Request) (wsConn *WsConnection, err error) {
	var conn *websocket.Conn
	if conn, err = Upgrader.Upgrade(resp, req, nil); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	wsConn = &WsConnection{
		conn:    conn,
		cancel:  cancel,
		lock:    sync.RWMutex{},
		inChan:  make(chan *WsMessage, 1000),
		outChan: make(chan *WsMessage, 1000),
		stoped:  false,
	}
	go wsConn.wsReadLoop(ctx)
	go wsConn.wsWriteLoop(ctx)
	return
}
