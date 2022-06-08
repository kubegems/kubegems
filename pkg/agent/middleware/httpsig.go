package middleware

import (
	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/service/handlers"
	"kubegems.io/kubegems/pkg/utils/httpsigs"
)

func SignerMiddleware() func(c *gin.Context) {
	signer := httpsigs.GetSigner()
	signer.AddWhiteList("/alert")
	signer.AddWhiteList("/alert")
	signer.AddWhiteList("/healthz")

	return func(c *gin.Context) {
		if err := signer.Validate(c.Request); err != nil {
			log.Error(err, "signer")
			handlers.Forbidden(c, err)
			c.Abort()
		}
	}
}
