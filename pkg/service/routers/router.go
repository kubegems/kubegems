package routers

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/server/define"
	"github.com/kubegems/gems/pkg/service/aaa"
	auth "github.com/kubegems/gems/pkg/service/aaa/authentication"
	"github.com/kubegems/gems/pkg/service/handlers"
	alerthandler "github.com/kubegems/gems/pkg/service/handlers/alerts"
	applicationhandler "github.com/kubegems/gems/pkg/service/handlers/application"
	approveHandler "github.com/kubegems/gems/pkg/service/handlers/approve"
	appstorehandler "github.com/kubegems/gems/pkg/service/handlers/appstore"
	auditloghandler "github.com/kubegems/gems/pkg/service/handlers/auditlog"
	clusterhandler "github.com/kubegems/gems/pkg/service/handlers/cluster"
	environmenthandler "github.com/kubegems/gems/pkg/service/handlers/environment"
	eventhandler "github.com/kubegems/gems/pkg/service/handlers/event"
	loginhandler "github.com/kubegems/gems/pkg/service/handlers/login"
	logqueryhandler "github.com/kubegems/gems/pkg/service/handlers/logquery"
	lokiloghandler "github.com/kubegems/gems/pkg/service/handlers/lokilog"
	messagehandler "github.com/kubegems/gems/pkg/service/handlers/message"
	"github.com/kubegems/gems/pkg/service/handlers/metrics"
	microservice "github.com/kubegems/gems/pkg/service/handlers/microservice"
	myinfohandler "github.com/kubegems/gems/pkg/service/handlers/myinfo"
	noproxyhandler "github.com/kubegems/gems/pkg/service/handlers/noproxy"
	projecthandler "github.com/kubegems/gems/pkg/service/handlers/project"
	proxyhandler "github.com/kubegems/gems/pkg/service/handlers/proxy"
	registryhandler "github.com/kubegems/gems/pkg/service/handlers/registry"
	sel "github.com/kubegems/gems/pkg/service/handlers/sels"
	systemhandler "github.com/kubegems/gems/pkg/service/handlers/system"
	systemrolehandler "github.com/kubegems/gems/pkg/service/handlers/systemrole"
	tenanthandler "github.com/kubegems/gems/pkg/service/handlers/tenant"
	userhandler "github.com/kubegems/gems/pkg/service/handlers/users"
	workloadreshandler "github.com/kubegems/gems/pkg/service/handlers/workloadres"
	"github.com/kubegems/gems/pkg/service/options"
	"github.com/kubegems/gems/pkg/utils/prometheus/collector"
	"github.com/kubegems/gems/pkg/version"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func getClientIP(c *gin.Context) string {
	forwardHeader := c.Request.Header.Get("x-forwarded-for")
	if len(forwardHeader) > 0 {
		firstAddress := strings.Split(forwardHeader, ",")[0]
		if net.ParseIP(strings.TrimSpace(firstAddress)) != nil {
			return firstAddress
		}
	}
	return c.ClientIP()
}

func RealClientIPMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		_, port, _ := net.SplitHostPort(strings.TrimSpace(ctx.Request.RemoteAddr))
		ip := getClientIP(ctx)
		ctx.Request.RemoteAddr = fmt.Sprintf("%s:%s", ip, port)
		ctx.Next()
	}
}

func NewRouter(_ *options.Options) *gin.Engine {
	router := gin.New()
	url := ginSwagger.URL(
		"/swagger/doc.json",
	)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))

	dir, _ := os.Getwd()
	router.StaticFS("/lokiExport", http.Dir(dir+"/lokiExport"))
	router.Use(
		// 请求数统计
		collector.GetRequestCollector().HandlerFunc(),
		// 日志
		log.DefaultGinLoggerMideare(),
		// Panic处理
		gin.Recovery(),
	)
	return router
}

func RegistRouter(router *gin.Engine, opts *options.Options, server define.ServerInterface, middlewares ...func(*gin.Context)) error {
	// 健康检测，版本
	router.Use(RealClientIPMiddleware())
	router.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "healthy"}) })
	router.GET("/version", func(c *gin.Context) { c.JSON(http.StatusOK, version.Get()) })
	router.GET("/v1/version", func(c *gin.Context) { handlers.OK(c, version.Get()) })

	// 登录和认证相关
	authMiddleware, err := auth.InitAuthMiddleware(server.GetOptions().System, server.GetDataBase(), server.GetRedis(), aaa.NewUserInfoHandler())
	if err != nil {
		return err
	}
	oauth := loginhandler.OAuthHandler{Midware: authMiddleware}
	router.POST("/v1/login", oauth.LoginHandler)
	router.GET("/v1/oauth/addr", oauth.GetOauthAddr)
	router.GET("/v1/oauth/callback", oauth.GetOauthToken)

	rg := router.Group("v1")

	// 注册中间件
	for _, mw := range middlewares {
		rg.Use(mw)
	}
	// 选项
	systemHandler := systemhandler.SystemHandler{ServerInterface: server}
	systemHandler.RegistRouter(rg)

	// 用户
	userHandler := &userhandler.UserHandler{ServerInterface: server}
	userHandler.RegistRouter(rg)

	// 系统角色
	systemroleHandler := &systemrolehandler.SystemRoleHandler{ServerInterface: server}
	systemroleHandler.RegistRouter(rg)

	// 集群
	clusterHandler := &clusterhandler.ClusterHandler{ServerInterface: server}
	clusterHandler.RegistRouter(rg)

	// 审计
	auditlogHandler := &auditloghandler.AuditLogHandler{ServerInterface: server}
	auditlogHandler.RegistRouter(rg)

	// 租户
	tenantHandler := &tenanthandler.TenantHandler{ServerInterface: server}
	tenantHandler.RegistRouter(rg)

	// 项目
	projectHandler := &projecthandler.ProjectHandler{ServerInterface: server}
	projectHandler.RegistRouter(rg)

	// 消息
	messageHandler := &messagehandler.MessageHandler{ServerInterface: server}
	messageHandler.RegistRouter(rg)

	// 消息
	approveHandler := &approveHandler.ApproveHandler{ServerInterface: server}
	approveHandler.RegistRouter(rg)

	// 事件
	eventHandler := &eventhandler.EventHandler{ServerInterface: server}
	eventHandler.RegistRouter(rg)

	// 告警规则
	alertRuleHandler := &alerthandler.AlertsHandler{ServerInterface: server, ClientSet: server.GetAgentsClientSet()}
	alertRuleHandler.RegistRouter(rg)

	metricsHandler := &metrics.MonitorHandler{ServerInterface: server}
	metricsHandler.RegistRouter(rg)

	// 告警
	alertmgrHandler := &alerthandler.AlertmanagerConfigHandler{ServerInterface: server}
	alertmgrHandler.RegistRouter(rg)

	// 环境
	environmentHandler := &environmenthandler.EnvironmentHandler{ServerInterface: server}
	environmentHandler.RegistRouter(rg)

	// 当前个人信息
	myHandler := &myinfohandler.MyHandler{ServerInterface: server}
	myHandler.RegistRouter(rg)

	// 应用商店
	appstoreHandler := &appstorehandler.AppstoreHandler{ServerInterface: server, AppStoreOpt: opts.Appstore}
	appstoreHandler.RegistRouter(rg)

	// 镜像仓库
	registryHandler := &registryhandler.RegistryHandler{ServerInterface: server}
	registryHandler.RegistRouter(rg)

	// 日志
	lokilogHandler := &lokiloghandler.LogHandler{ServerInterface: server}
	lokilogHandler.RegistRouter(rg)

	// 日志查询历史
	logqueryhistoryHandler := &logqueryhandler.LogQueryHistoryHandler{ServerInterface: server}
	logqueryhistoryHandler.RegistRouter(rg)

	// 日志查询快照
	logquerysnapshotHandler := &logqueryhandler.LogQuerySnapshotHandler{ServerInterface: server}
	logquerysnapshotHandler.RegistRouter(rg)

	// 非反向代理资源[HPA]
	hpaHandler := &noproxyhandler.HpaHandler{ServerInterface: server}
	hpaHandler.RegistRouter(rg)

	// 非反向代理资源[PVC]
	pvcHandler := &noproxyhandler.PersistentVolumeClaimHandler{ServerInterface: server}
	pvcHandler.RegistRouter(rg)

	// 非反向代理资源[卷快照]
	volumeSnapshotHandler := &noproxyhandler.VolumeSnapshotHandler{ServerInterface: server}
	volumeSnapshotHandler.RegistRouter(rg)

	//  资源统计相关
	workloadHandler := &workloadreshandler.WorkloadHandler{ServerInterface: server}
	workloadHandler.RegistRouter(rg)

	// sels
	selHandler := &sel.SelsHandler{ServerInterface: server}
	selHandler.RegistRouter(rg)

	// app handler
	appHandler := applicationhandler.MustNewApplicationDeployHandler(server)
	appHandler.RegistRouter(rg)

	// microservice  handler
	microservicehandler := microservice.NewMicroServiceHandler(server)
	microservicehandler.RegistRouter(rg)

	// workload 的反向代理
	proxyHandler := proxyhandler.NewProxyHandler(server)
	rg.Any("/proxy/cluster/:cluster/*action", proxyHandler.Proxy)
	router.Any("/v1/service-proxy/cluster/:cluster/namespace/:namespace/service/:service/port/:port/*action", proxyHandler.ProxyService)

	return nil
}
