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
	ResizeEvent chan *remotecommand.TerminalSize
}

type xtermMessage struct {
	MsgType string `json:"type"`
	Input   string `json:"input"`
	Rows    uint16 `json:"rows"`
	Cols    uint16 `json:"cols"`
}

// Next must return nil if the connection closed
func (handler *StreamHandler) Next() (size *remotecommand.TerminalSize) {
	ret := <-handler.ResizeEvent
	return ret
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
		handler.ResizeEvent <- &remotecommand.TerminalSize{Width: xtermMsg.Cols, Height: xtermMsg.Rows}
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
