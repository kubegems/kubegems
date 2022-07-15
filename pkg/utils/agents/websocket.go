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

package agents

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type WebsocketRoundTripper struct {
	Dialer *websocket.Dialer
	Result chan ConnInitResult
}
type ConnInitResult struct {
	Conn *websocket.Conn
	Resp *http.Response
	Err  error
}

func NewWebsocketRoundTripper(dialer *websocket.Dialer) *WebsocketRoundTripper {
	return &WebsocketRoundTripper{
		Dialer: dialer,
		Result: make(chan ConnInitResult, 1),
	}
}

func (d *WebsocketRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	conn, resp, err := d.Dialer.Dial(r.URL.String(), r.Header)
	d.Result <- ConnInitResult{
		Conn: conn,
		Resp: resp,
		Err:  err,
	}
	if err == nil {
		defer conn.Close()
	}
	ch := make(chan bool, 1)
	conn.SetCloseHandler(func(code int, text string) error {
		message := websocket.FormatCloseMessage(code, "")
		conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		ch <- true
		return nil
	})
	<-ch
	return resp, err
}
