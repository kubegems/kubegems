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
