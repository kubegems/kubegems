// Copyright 2023 The kubegems.io Authors
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

package auth

import (
	"io"
	"net/http"
	"strings"

	"golang.org/x/exp/slices"
)

func HttpHeaderToMap(header http.Header) map[string]string {
	m := make(map[string]string)
	for k, v := range header {
		m[k] = strings.Join(v, ",")
	}
	return m
}

func ReadBodySafely(req *http.Request, allowsContentType []string, maxReadSize int) []byte {
	contenttype, contentlen := req.Header.Get("Content-Type"), req.ContentLength
	if contenttype == "" || contentlen == 0 {
		return nil
	}
	allowed := slices.ContainsFunc(allowsContentType, func(s string) bool {
		return strings.HasPrefix(contenttype, s)
	})
	if !allowed {
		return nil
	}
	cachesize := maxReadSize
	if contentlen < int64(maxReadSize) {
		cachesize = int(contentlen)
	}
	if cachesize <= 0 {
		return nil
	}
	cachedbody := make([]byte, cachesize)
	n, err := io.ReadFull(req.Body, cachedbody)
	// io.ReadFull returns io.ErrUnexpectedEOF if EOF is encountered before filling the buffer.
	if err != nil && err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	req.Body = NewCachedBody(req.Body, cachedbody[:n], err)
	return cachedbody[:n]
}

var _ io.ReadCloser = &CachedBody{}

type CachedBody struct {
	cached []byte
	err    error // early read error
	readn  int
	body   io.ReadCloser
}

// NewCachedBody returns a new CachedBody.
// a CachedBody is a io.ReadCloser that read from cached first, then read from body.
func NewCachedBody(body io.ReadCloser, cached []byte, earlyerr error) *CachedBody {
	return &CachedBody{body: body, cached: cached, err: earlyerr}
}

func (w *CachedBody) Read(p []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	if w.readn < len(w.cached) {
		n += copy(p, w.cached[w.readn:])
		w.readn += n
		if n == len(p) {
			return n, nil
		}
		p = p[n:] // continue read from body
	}
	bn, err := w.body.Read(p)
	n += bn
	return n, err
}

func (w *CachedBody) Close() error {
	return w.body.Close()
}

var _ http.ResponseWriter = &StatusResponseWriter{}

type StatusResponseWriter struct {
	Inner        http.ResponseWriter
	Code         int
	Cache        []byte
	maxCacheSize int
}

func NewStatusResponseWriter(inner http.ResponseWriter, maxCacheBodySize int) *StatusResponseWriter {
	return &StatusResponseWriter{Inner: inner, maxCacheSize: maxCacheBodySize}
}

func (w *StatusResponseWriter) Header() http.Header {
	return w.Inner.Header()
}

func (w *StatusResponseWriter) Write(p []byte) (n int, err error) {
	if w.Code == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if len(w.Cache) < w.maxCacheSize {
		w.Cache = append(w.Cache, p...)
	}
	return w.Inner.Write(p)
}

func (w *StatusResponseWriter) WriteHeader(statusCode int) {
	w.Code = statusCode
	w.Inner.WriteHeader(statusCode)
}
