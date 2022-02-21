package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"kubegems.io/pkg/utils/httputil"
)

func (c TypedClient) DialWebsocket(ctx context.Context, path string, headers ...http.Header) (*websocket.Conn, *http.Response, error) {
	wsu := (&url.URL{
		Scheme: func() string {
			if c.BaseAddr.Scheme == "http" {
				return "ws"
			} else {
				return "wss"
			}
		}(),
		Host: c.BaseAddr.Host,
		Path: c.BaseAddr.Path + "/" + path,
	}).String()

	if len(headers) > 0 {
		return c.websocket.DialContext(ctx, wsu, headers[0])
	} else {
		return c.websocket.DialContext(ctx, wsu, nil)
	}
}

type Request struct {
	Method  string
	Path    string // queries 可以放在 path 中
	Query   url.Values
	Headers http.Header
	Body    interface{}
	Into    interface{}
}

func QueryFrom(kvs map[string]string) url.Values {
	value := url.Values{}
	for k, v := range kvs {
		value.Add(k, v)
	}
	return value
}

func HeadersFrom(kvs map[string]string) http.Header {
	header := http.Header{}
	for k, v := range kvs {
		header.Add(k, v)
	}
	return header
}

func WrappedResponse(intodata interface{}) *httputil.Response {
	return &httputil.Response{Data: intodata}
}

func (c TypedClient) DoRawRequest(ctx context.Context, clientreq Request) (*http.Response, error) {
	addr := c.BaseAddr.String() + clientreq.Path

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
	if clientreq.Headers.Get("Content-Type") == "" {
		req.Header.Add("Content-Type", "application/json")
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

func (c TypedClient) DoRequest(ctx context.Context, req Request) error {
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
	if req.Into != nil {
		if err := json.NewDecoder(resp.Body).Decode(req.Into); err != nil {
			return fmt.Errorf("decode resp: err: %w", err)
		}
	}
	return nil
}
