package api

import (
	"context"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/msgbus/options"
	"kubegems.io/pkg/msgbus/switcher"
	"kubegems.io/pkg/service/aaa"
	auth "kubegems.io/pkg/service/aaa/authentication"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/redis"
)

func NewGinServer(opts *options.Options, database *database.Database, redis *redis.Client, ms *switcher.MessageSwitcher) (*gin.Engine, error) {
	r := gin.Default()
	// 初始化需要注册的中间件
	authmidware, err := auth.InitAuthMiddleware(opts.System, database, redis, aaa.NewUserInfoHandler())
	if err != nil {
		return nil, err
	}
	middlewares := []func(*gin.Context){
		authmidware.MiddlewareFunc(),
	}

	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"healthy": "ok"}) })
	for _, md := range middlewares {
		r.Use(md)
	}
	rg := r.Group("/v2")
	msgHandler := &MessageHandler{
		UserInfoHandler: aaa.NewUserInfoHandler(),
		Switcher:        ms,
	}
	msgHandler.RegistRouter(rg)
	return r, nil
}

func RunGinServer(ctx context.Context, options *options.Options, db *database.Database, redis *redis.Client, ms *switcher.MessageSwitcher) error {
	r, err := NewGinServer(options, db, redis, ms)
	if err != nil {
		return err
	}
	httpserver := &http.Server{
		Addr:    options.System.Listen,
		Handler: r,
		BaseContext: func(l net.Listener) context.Context {
			return ctx // 注入basecontext
		},
	}
	go func() {
		<-ctx.Done()
		httpserver.Close()
	}()
	log.FromContextOrDiscard(ctx).Info("start listen", "addr", httpserver.Addr)
	return httpserver.ListenAndServe()
}
