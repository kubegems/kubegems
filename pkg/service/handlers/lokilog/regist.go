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

package lokilog

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type LogHandler struct {
	base.BaseHandler
}

func (h *LogHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/log/:cluster_name/queryrange", h.QueryRange)
	rg.GET("/log/:cluster_name/labels", h.Labels)
	rg.GET("/log/:cluster_name/export", h.Export)
	rg.GET("/log/:cluster_name/label/:label/values", h.LabelValues)
	rg.GET("/log/:cluster_name/querylanguage", h.QueryLanguage)
	rg.GET("/log/:cluster_name/series", h.Series)
	rg.GET("/log/:cluster_name/context", h.Context)
}
