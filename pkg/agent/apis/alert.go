package apis

import (
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/msgbus"
)

// 获取各个集群的告警信息
type AlertHandler struct {
	*Watcher
}

// @Tags         Agent.V1
// @Summary      检查alertmanagerconfig
// @Description  检查alertmanagerconfig
// @Accept       json
// @Produce      json
// @Success      200  {object}  handlers.ResponseStruct{Data=string}  ""
// @Router       /alert [post]
// @Security     JWT
func (h *AlertHandler) Webhook(c *gin.Context) {
	b, _ := io.ReadAll(c.Request.Body)
	msg := msgbus.NotifyMessage{
		MessageType: msgbus.Alert,
		Content:     string(b),
	}
	errs := h.Watcher.send(msg)
	if len(errs) != 0 {
		log.Errorf("send error: %v", errs)
		NotOK(c, fmt.Errorf(strings.Join(errs, ",")))
		return
	}
	OK(c, errs)
}
