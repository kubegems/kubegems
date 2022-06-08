package ws

import (
	"encoding/json"
	"io"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
	"kubegems.io/kubegems/pkg/log"
)

type StreamHandler struct {
	WsConn      *WsConnection
	ResizeEvent chan remotecommand.TerminalSize
}

type xtermMessage struct {
	MsgType string `json:"type"`
	Input   string `json:"input"`
	Rows    uint16 `json:"rows"`
	Cols    uint16 `json:"cols"`
}

func (handler *StreamHandler) Next() (size *remotecommand.TerminalSize) {
	ret := <-handler.ResizeEvent
	size = &ret
	return
}

func (handler *StreamHandler) Read(p []byte) (size int, err error) {
	var (
		msg      *WsMessage
		xtermMsg xtermMessage
	)
	if msg, err = handler.WsConn.WsRead(); err != nil {
		log.Error(err, "read websocket")
		handler.WsConn.WsClose()
		return
	}
	if msg == nil {
		return
	}
	if err = json.Unmarshal([]byte(msg.Data), &xtermMsg); err != nil {
		log.Error(err, "unmarshal websocket message")
		return
	}
	switch xtermMsg.MsgType {
	case "resize":
		handler.ResizeEvent <- remotecommand.TerminalSize{Width: xtermMsg.Cols, Height: xtermMsg.Rows}
	case "input":
		size = len(xtermMsg.Input)
		copy(p, xtermMsg.Input)
	case "close":
		handler.WsConn.WsClose()
		err = io.EOF
	}
	return
}

func (handler *StreamHandler) Write(p []byte) (size int, err error) {
	copyData := make([]byte, len(p))
	copy(copyData, p)
	size = len(copyData)
	err = handler.WsConn.WsWrite(websocket.TextMessage, validUTF8(copyData))
	if err != nil {
		log.Error(err, "write websocket")
		handler.WsConn.WsClose()
	}
	return
}

func validUTF8(arr []byte) []byte {
	ret := []rune{}
	for len(arr) > 0 {
		r, size := utf8.DecodeRune(arr)
		arr = arr[size:]
		ret = append(ret, r)
	}
	return []byte(string(ret))
}
