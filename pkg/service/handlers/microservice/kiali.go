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

package microservice

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/utils/httputil/response"
)

type KialiAPIRequest struct {
	Path           string
	VirtualspaceId string
}

// @Tags        VirtualSpace
// @Summary     kiali代理
// @Description kiali api 代理
// @Accept      json
// @Produce     json
// @Param       virtualspace_id path     uint   true "virtualspace_id"
// @Param       environment_id  path     uint   true "environment_id（通过环境寻找目标集群）"
// @Param       path            path     string true "访问 kiali service 的路径"
// @Success     200             {object} object "kiali 原始响应"
// @Router      /v1/virtualspace/{virtualspace_id}/environment/environment_id/kiali/{kiaklipath} [get]
// @Security    JWT
func (h *VirtualSpaceHandler) KialiAPI(c *gin.Context) {
	options := h.MicroserviceOptions
	kialisvc, kialinamespace := options.KialiName, options.KialiNamespace
	// get and check env
	env := models.Environment{}
	ctx := c.Request.Context()
	if err := h.GetDB().WithContext(ctx).Preload("Cluster").First(&env, c.Param("environment_id")).Error; err != nil {
		handlers.NotOK(c, err)
		return
	}
	cluster, kialipath := env.Cluster.ClusterName, c.Param("path")
	cli, err := h.clientOf(ctx, cluster)
	if err != nil {
		handlers.NotOK(c, err)
		return
	}
	w, r := c.Writer, c.Request
	dest := &url.URL{
		Scheme: "http",
		Host:   kialisvc + "." + kialinamespace + ":20001",
	}
	r.URL.Path = "/kiali" + kialipath // rewrite path
	rp := cli.ReverseProxy(dest)
	rp.ModifyResponse = ResponseBodyRewriter(func(src io.Reader, dst io.Writer) error {
		var data interface{}
		if err := json.NewDecoder(src).Decode(&data); err != nil {
			return err
		}
		// wrap kiakli response with our response.Rsponse
		return json.NewEncoder(dst).Encode(response.Response{Data: data})
	})
	rp.ServeHTTP(w, r)
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
