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
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func (c TypedClient) Proxy(ctx context.Context, obj client.Object, port int, req *http.Request, writer http.ResponseWriter, rewritefunc func(r *http.Response) error) error {
	gvk, err := apiutil.GVKForObject(obj, c.RuntimeScheme)
	if err != nil {
		return err
	}

	if gvk.Kind != "Service" && gvk.Kind != "Pod" {
		return fmt.Errorf("unsupported proxy for %s", gvk.GroupKind().String())
	}

	addr := fmt.Sprintf("%s/internal/core/v1/namespaces/%s/%s/%s:%d/proxy",
		c.BaseAddr.String(),
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
		Transport:      c.HTTPClient.Transport,
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
