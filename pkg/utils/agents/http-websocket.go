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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

func (c DelegateClient) DialWebsocket(ctx context.Context, rpath string, headers http.Header) (*websocket.Conn, *http.Response, error) {
	wsu := (&url.URL{
		Scheme: func() string {
			if c.BaseAddr().Scheme == "http" {
				return "ws"
			} else {
				return "wss"
			}
		}(),
		Host: c.BaseAddr().Host,
		Path: path.Join(c.BaseAddr().Path, rpath),
	}).String()
	return c.websocket.DialContext(ctx, wsu, headers)
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

func WrappedResponse(intodata interface{}) *response.Response {
	return &response.Response{Data: intodata}
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

	// inject for propagator to do distribute tracing
	otel.GetTextMapPropagator().Inject(req.Context(), propagation.HeaderCarrier(req.Header))

	// queries
	query := req.URL.Query()
	for k, vs := range clientreq.Query {
		for _, v := range vs {
			query.Add(k, v)
		}
	}
	req.URL.RawQuery = query.Encode()

	return c.HTTPClient.Do(req)
}

func (c TypedClient) DoRequest(ctx context.Context, req Request) error {
	if req.Method == "" {
		req.Method = "GET"
	}
	ctx, span := tracer.Start(ctx,
		fmt.Sprintf("TypedClient.%s %s", req.Method, req.Path),
		trace.WithAttributes(
			attribute.String("k8s.apiserver.host", c.BaseAddr.Host),
			attribute.String("request.method", req.Method),
			attribute.String("request.path", req.Path),
			attribute.String("request.query", req.Query.Encode()),
		),
	)
	defer span.End()
	resp, err := c.DoRawRequest(ctx, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer resp.Body.Close()

	// err
	if resp.StatusCode >= http.StatusBadRequest {
		content, _ := io.ReadAll(resp.Body) // resp body may be empty
		err := fmt.Errorf("request error: code %d, body %s", resp.StatusCode, string(content))
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	// success
	if req.Into != nil {
		if err := json.NewDecoder(resp.Body).Decode(req.Into); err != nil {
			err := fmt.Errorf("decode resp: err: %w", err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	span.SetStatus(codes.Ok, "")
	return nil
}
