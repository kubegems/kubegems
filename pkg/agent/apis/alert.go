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
	"io"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/utils/msgbus"
)

// 获取各个集群的告警信息
type AlertHandler struct {
	*Watcher
}

//	@Tags			Agent.V1
//	@Summary		kubegems default alert webhook
//	@Description	kubegems default alert webhook
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	handlers.ResponseStruct{Data=string}	""
//	@Router			/alert [post]
//	@Security		JWT
func (h *AlertHandler) Webhook(c *gin.Context) {
	b, _ := io.ReadAll(c.Request.Body)
	msg := msgbus.NotifyMessage{
		MessageType: msgbus.Alert,
		Content:     string(b),
	}
	h.Watcher.DispatchMessage(msg)
	OK(c, nil)
}
