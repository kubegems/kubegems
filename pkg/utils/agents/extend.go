package agents

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/proxy"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func (c *TypedClient) DialWebsocket(ctx context.Context, path string, headers ...http.Header) (*websocket.Conn, *http.Response, error) {
	wsu := (&url.URL{
		Scheme: func() string {
			if c.serveraddr.Scheme == "http" {
				return "ws"
			} else {
				return "wss"
			}
		}(),
		Host: c.serveraddr.Host,
		Path: c.serveraddr.Path + "/" + path,
	}).String()

	if len(headers) > 0 {
		return c.websocket.DialContext(ctx, wsu, headers[0])
	} else {
		return c.websocket.DialContext(ctx, wsu, nil)
	}
}

func (c *TypedClient) DoRawRequest(ctx context.Context, clientreq Request) (*http.Response, error) {
	addr := c.serveraddr.String() + clientreq.Path

	var body io.Reader

	switch clientreqbody := clientreq.Body.(type) {
	case []byte:
		body = bytes.NewReader(clientreqbody)
	case io.Reader:
		body = clientreqbody
	default:
		content, err := json.Marshal(clientreqbody)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(content)
	}

	req, err := http.NewRequestWithContext(ctx, clientreq.Method, addr, body)
	if err != nil {
		return nil, err
	}

	// headers
	for k, vs := range clientreq.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	// queries
	query := req.URL.Query()
	for k, vs := range clientreq.Query {
		for _, v := range vs {
			query.Add(k, v)
		}
	}
	req.URL.RawQuery = query.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *TypedClient) DoRequest(ctx context.Context, req Request) error {
	resp, err := c.DoRawRequest(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// err
	if resp.StatusCode >= http.StatusBadRequest {
		content, _ := io.ReadAll(resp.Body) // resp body may be empty
		return fmt.Errorf("request error: code %d, body %s", resp.StatusCode, string(content))
	}

	// success
	if err := json.NewDecoder(resp.Body).Decode(req.Into); err != nil {
		return err
	}
	return nil
}

func (c *TypedClient) Proxy(ctx context.Context, obj client.Object, port int, req *http.Request, writer http.ResponseWriter, rewritefunc func(r *http.Response) error) error {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return err
	}

	if gvk.Kind != "Service" && gvk.Kind != "Pod" {
		return fmt.Errorf("unsupported proxy for %s", gvk.GroupKind().String())
	}

	addr := fmt.Sprintf("%s/internal/core/v1/namespaces/%s/%s/%s:%d/proxy",
		c.serveraddr,
		obj.GetNamespace(),
		gvk.Kind,
		obj.GetName(),
		port,
	)
	target, err := url.Parse(addr)
	if err != nil {
		return err
	}

	(&httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = target.Path + req.URL.Path
			req.Host = target.Host
		},
		Transport:      c.http.Transport,
		ModifyResponse: rewritefunc,
	}).ServeHTTP(writer, req)
	return nil
}

// ResponseBodyRewriter 会正确处理 gzip 以及 deflate 的content-encodeing 以及response 的content-length
// 用于需要修改代理的响应体是非常有用
func ResponseBodyRewriter(rewritefunc func(io.Reader, io.Writer) error) func(resp *http.Response) error {
	return func(r *http.Response) error {
		reader := r.Body
		writer := &bytes.Buffer{}

		defer func() {
			r.Body.Close()
			r.Body = io.NopCloser(writer)
			r.ContentLength = int64(writer.Len())
			r.Header.Set("Content-Length", strconv.Itoa(writer.Len()))
		}()

		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			gzr, err := gzip.NewReader(reader)
			if err != nil {
				return err
			}
			gzw := gzip.NewWriter(writer)
			defer func() {
				gzw.Close()
				gzw.Flush()
			}()
			return rewritefunc(gzr, gzw)
		case "deflate":
			flw, err := flate.NewWriter(writer, 0)
			if err != nil {
				return err
			}
			defer func() {
				flw.Close()
				flw.Flush()
			}()
			return rewritefunc(flate.NewReader(reader), flw)
		default:
			return rewritefunc(reader, writer)
		}
	}
}

type PortForwarder struct {
	ctx            context.Context
	cancel         context.CancelFunc
	httpRequestUrl *url.URL
	httpreq        *http.Request
	ln             net.Listener
}

func newPortForwarder(ctx context.Context, target string) (*PortForwarder, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	p := &PortForwarder{
		ctx:            ctx,
		cancel:         cancel,
		httpRequestUrl: u,
		httpreq:        req,
	}

	if err := p.start(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PortForwarder) start() error {
	// listen
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return err
	}
	p.ln = ln

	log.Infof("port forwarder listening on %s", ln.Addr().String())

	go func() {
		defer ln.Close()
		for {
			conn, err := ln.Accept()
			select {
			case <-p.ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Errorf("error accepting connection: %v", err)
				continue
			}
			go p.servconn(conn)
		}
	}()

	return nil
}

func (p *PortForwarder) servconn(conn net.Conn) {
	dst, err := net.Dial("tcp", p.httpRequestUrl.Host)
	if err != nil {
		log.Errorf("error dial: %v", err)
		return
	}
	// open as http
	if err := p.httpreq.Write(dst); err != nil {
		log.Errorf("error wrilte request: %v", err)
		return
	}
	// using as tcp
	if err := proxy.CopyDuplex(conn, dst, -1); err != nil {
		log.Errorf("error copy duplex: %v", err)
	}
}

func (p *PortForwarder) ListenAddr() net.Addr {
	return p.ln.Addr()
}

func (p *PortForwarder) Stop() {
	p.cancel()
	p.ln.Close()
}

//  PortForward
// Deprecated: 无法使用，因 service 与 agent 中间还有一层 http proxy(apiserver). 无法直接使用 tcp 。
func (c *TypedClient) PortForward(ctx context.Context, obj client.Object, port int) (*PortForwarder, error) {
	gvk, err := apiutil.GVKForObject(obj, c.scheme)
	if err != nil {
		return nil, err
	}

	if gvk.Kind != "Service" && gvk.Kind != "Pod" {
		return nil, fmt.Errorf("unsupported port forwarding of %s", gvk.GroupKind().String())
	}

	queries := url.Values{}
	queries.Set("port", strconv.Itoa(port))

	addr := fmt.Sprintf("%s/internal/core/v1/namespaces/%s/%s/%s/portforward?%s",
		c.serveraddr,
		obj.GetNamespace(),
		gvk.Kind,
		obj.GetName(),
		queries.Encode(),
	)
	forwarder, err := newPortForwarder(ctx, addr)
	if err != nil {
		return nil, err
	}
	return forwarder, nil
}
