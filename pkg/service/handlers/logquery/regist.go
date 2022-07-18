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

package logqueryhandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

type LogQuerySnapshotHandler struct {
	base.BaseHandler
}

type LogQueryHistoryHandler struct {
	base.BaseHandler
}

func (h *LogQueryHistoryHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/logqueryhistory", h.ListLogQueryHistory)
	rg.POST("/logqueryhistory", h.PostLogQueryHistory)
	rg.DELETE("/logqueryhistory", h.BatchDeleteLogQueryHistory)
	rg.DELETE("/logqueryhistory/:logqueryhistory_id",
		h.DeleteLogQueryHistory)
}

func (h *LogQuerySnapshotHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/logquerysnapshot", h.ListLogQuerySnapshot)
	rg.GET("/logquerysnapshot/:logquerysnapshot_id", h.RetrieveLogQuerySnapshot)
	rg.DELETE("/logquerysnapshot/:logquerysnapshot_id",
		h.DeleteLogQuerySnapshot)
	rg.POST("/logquerysnapshot",
		h.PostLogQuerySnapshot)
}
