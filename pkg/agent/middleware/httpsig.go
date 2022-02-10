package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/service/handlers"
	"github.com/kubegems/gems/pkg/utils/httpsigs"
)

func SignerMiddleware() func(c *gin.Context) {
	signer := httpsigs.GetSigner()
	signer.AddWhiteList("/alert")
	signer.AddWhiteList("/alert")
	signer.AddWhiteList("/healthz")

	return func(c *gin.Context) {
		if err := signer.Validate(c.Request); err != nil {
			handlers.Forbidden(c, err)
			c.Abort()
		}
	}
}
