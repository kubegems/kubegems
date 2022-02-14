package proxy

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/models"
)

type Msg struct {
	MsgType int
	Content []byte
}

type xtermMessage struct {
	MsgType string `json:"type"`  // 类型:resize客户端调整终端, input客户端输入
	Input   string `json:"input"` // msgtype=input情况下使用
	Rows    uint16 `json:"rows"`  // msgtype=resize情况下使用
	Cols    uint16 `json:"cols"`  // msgtype=resize情况下使用
}

func Transport(local, proxy *websocket.Conn, c *gin.Context, user *models.User, auditFunc func(string)) {
	p := WebSocketProxy{
		RequestContext: c,
		Source:         local,
		Target:         proxy,
		SourceChan:     make(chan Msg, 100),
		TargetChan:     make(chan Msg, 100),
		Done:           make(chan bool, 2),

		UserName: user.Username,
		buf:      bytes.NewBuffer([]byte{}),
	}

	p.AuditFunc = auditFunc
	p.proxy()
}

type WebSocketProxy struct {
	RequestContext *gin.Context
	Source         *websocket.Conn
	Target         *websocket.Conn
	SourceChan     chan Msg
	TargetChan     chan Msg
	Done           chan bool
	UserName       string
	AuditFunc      func(string)
	buf            *bytes.Buffer
}

func (wsp *WebSocketProxy) audit(msg []byte) {
	tmsg := xtermMessage{}
	_ = json.Unmarshal(msg, &tmsg)
	if tmsg.MsgType != "input" {
		return
	}
	// 不知道为什么要发这个，eg. \u001b[19;5R    \u001b[7;5R
	// ESC按键UNICODE
	if strings.Contains(tmsg.Input, "\u001b") {
		return
	}

	bts := []byte(tmsg.Input)
	if bytes.ContainsAny(bts, "\r") {
		bts = bytes.Trim(bts, "\r")
		wsp.buf.Write(bts)
		if wsp.AuditFunc != nil {
			wsp.AuditFunc(wsp.buf.String())
		}
		wsp.buf.Reset()
	} else {
		wsp.buf.Write(bts)
	}
}

func (wsp *WebSocketProxy) sourceRead() {
	for {
		msgtype, msg, e := wsp.Source.ReadMessage()
		if e != nil {
			log.Errorf("failed to read message from ws %v", e)
			wsp.Done <- true
			return
		}
		go wsp.audit(msg)

		wsp.SourceChan <- Msg{msgtype, msg}
	}
}

func (wsp *WebSocketProxy) targetRead() {
	for {
		lt, lmsg, err := wsp.Target.ReadMessage()
		if err != nil {
			wsp.Done <- true
			return
		}
		wsp.TargetChan <- Msg{lt, lmsg}
	}
}

func (wsp *WebSocketProxy) proxy() {
	go wsp.sourceRead()
	go wsp.targetRead()
	for {
		select {
		case msg := <-wsp.SourceChan:
			if e := wsp.Target.WriteMessage(msg.MsgType, msg.Content); e != nil {
				wsp.Done <- true
			}
		case msg := <-wsp.TargetChan:
			if e := wsp.Source.WriteMessage(msg.MsgType, msg.Content); e != nil {
				wsp.Done <- true
			}
		case <-wsp.Done:
			wsp.Target.Close()
			wsp.Source.Close()
			return
		}
	}
}
