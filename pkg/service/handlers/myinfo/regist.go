package myinfohandler

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/service/handlers/base"
)

type MyHandler struct {
	base.BaseHandler
}

func (h *MyHandler) RegistRouter(rg *gin.RouterGroup) {
	rg.GET("/my/info", h.Myinfo)
	rg.GET("/my/auth", h.MyAuthority)
	rg.GET("/my/tenants", h.MyTenants)
	rg.POST("/my/reset_password", h.ResetPassword)

	rg.GET("/my/config/:name", h.GetNormalConfig)
}
