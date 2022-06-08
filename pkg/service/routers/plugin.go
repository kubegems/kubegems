//go:build !internal
// +build !internal

package routers

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/service/handlers/base"
)

func registPlugins(rg *gin.RouterGroup, basehandler base.BaseHandler) error {
	return nil
}
