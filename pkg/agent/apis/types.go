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

package apis

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/library/rest/response"
)

// 如果是非成功的响应，使用 NotOK
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, response.Response{Message: "ok", Data: data})
}

func NotOK(c *gin.Context, err error) {
	NotOKCode(c, http.StatusBadRequest, err)
}

func NotOKCode(c *gin.Context, code int, err error) {
	log.Errorf("notok: %v", err)
	statusCode := http.StatusBadRequest
	// 增加针对 apierrors 状态码适配
	statuserr := &apierrors.StatusError{}
	if errors.As(err, &statuserr) {
		c.AbortWithStatusJSON(int(statuserr.Status().Code), response.Response{Message: err.Error(), Error: statuserr})
		return
	}
	c.AbortWithStatusJSON(statusCode, response.Response{Message: err.Error(), Error: err})
}
