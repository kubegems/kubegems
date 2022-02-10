package apis

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/service/handlers"
)

// 如果是非成功的响应，使用 NotOK
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, handlers.ResponseStruct{Message: "ok", Data: data})
}

func NotOK(c *gin.Context, err error) {
	log.Errorf("notok: %v", err)
	statusCode := http.StatusBadRequest

	// 增加针对 apierrors 状态码适配
	statuserr := &apierrors.StatusError{}
	if errors.As(err, &statuserr) {
		c.AbortWithStatusJSON(int(statuserr.Status().Code), handlers.ResponseStruct{Message: err.Error(), ErrorData: statuserr})
		return
	}

	c.AbortWithStatusJSON(statusCode, handlers.ResponseStruct{Message: err.Error(), ErrorData: err})
}
