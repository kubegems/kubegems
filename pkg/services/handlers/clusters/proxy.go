package clusterhandler

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"path"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/gorilla/websocket"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/services/handlers"
	"kubegems.io/pkg/utils/agents"
)

const (
	AgentModeApiServer = "apiServerProxy"
	AgentModeAHTTP     = "http"
	AgentModeHTTPS     = "https"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handler) Proxy(req *restful.Request, resp *restful.Response) {
	req.Request.Header.Del("Authorization")
	if websocket.IsWebSocketUpgrade(req.Request) {
		h.ProxyWebsocket(req, resp)
	} else {
		h.ProxyHTTP(req, resp)
	}
}

func (h *Handler) ProxyHTTP(req *restful.Request, resp *restful.Response) {
	cluster := req.PathParameter("cluster")
	v, err := h.Agents().ClientOf(req.Request.Context(), cluster)
	if err != nil {
		log.Error(err, "failed to load agent client", "cluster", cluster)
		handlers.BadRequest(resp, err)
		return
	}
	h.ReverseProxyOn(v).ServeHTTP(resp, req.Request)
}

func (h *Handler) ReverseProxyOn(cli agents.Client) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Path = getTargetPath(cli.Name(), req)
		},
		Transport: RoundTripOf(cli),
	}
}

// RoundTripOf
func RoundTripOf(cli agents.Client) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return cli.DoRawRequest(req.Context(), agents.Request{
			Method:  req.Method,
			Path:    req.URL.Path,
			Query:   req.URL.Query(),
			Headers: req.Header,
			Body:    req.Body,
		})
	})
}

type RoundTripperFunc func(req *http.Request) (*http.Response, error)

func (c RoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return c(req)
}

func (h *Handler) ProxyWebsocket(req *restful.Request, resp *restful.Response) {
	cluster := req.PathParameter("cluster")
	proxyPath := req.PathParameter("action")

	v, err := h.Agents().ClientOf(req.Request.Context(), cluster)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}

	header := http.Header{}
	for key, values := range req.Request.URL.Query() {
		header.Add(key, strings.Join(values, ","))
	}

	proxyConn, wresp, err := v.DialWebsocket(req.Request.Context(), proxyPath, header)
	if err != nil {
		resp.WriteHeader(wresp.StatusCode)
		resp.Write([]byte(err.Error()))
		return
	}
	localConn, err := upgrader.Upgrade(resp.ResponseWriter, req.Request, nil)
	if err != nil {
		handlers.BadRequest(resp, err)
		return
	}
	Transport(localConn, proxyConn)
}

func getTargetPath(name string, req *http.Request) (realpath string) {
	prefix := path.Join("/v2/clusters", name, "proxy")
	trimed := strings.TrimPrefix(req.URL.Path, prefix)
	if strings.HasPrefix(trimed, "/custom") {
		return trimed
	} else {
		return "/v1" + trimed
	}
}

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

func Transport(local, proxy *websocket.Conn) {
	p := WebSocketProxy{
		Source:     local,
		Target:     proxy,
		SourceChan: make(chan Msg, 100),
		TargetChan: make(chan Msg, 100),
		Done:       make(chan bool, 2),

		buf: bytes.NewBuffer([]byte{}),
	}

	p.proxy()
}

type WebSocketProxy struct {
	Source     *websocket.Conn
	Target     *websocket.Conn
	SourceChan chan Msg
	TargetChan chan Msg
	Done       chan bool
	UserName   string
	AuditFunc  func(string)
	buf        *bytes.Buffer
}

func (wsp *WebSocketProxy) sourceRead() {
	for {
		msgtype, msg, err := wsp.Source.ReadMessage()
		if err != nil {
			log.Errorf("failed to read message from source ws %v", err)
			wsp.Done <- true
			return
		}

		wsp.SourceChan <- Msg{msgtype, msg}
	}
}

func (wsp *WebSocketProxy) targetRead() {
	for {
		lt, lmsg, err := wsp.Target.ReadMessage()
		if err != nil {
			log.Errorf("failed to read message from target ws %v", err)
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
