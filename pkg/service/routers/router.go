package routers

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/aaa"
	auth "kubegems.io/pkg/service/aaa/authentication"
	"kubegems.io/pkg/service/handlers"
	alerthandler "kubegems.io/pkg/service/handlers/alerts"
	applicationhandler "kubegems.io/pkg/service/handlers/application"
	approveHandler "kubegems.io/pkg/service/handlers/approve"
	appstorehandler "kubegems.io/pkg/service/handlers/appstore"
	auditloghandler "kubegems.io/pkg/service/handlers/auditlog"
	"kubegems.io/pkg/service/handlers/base"
	clusterhandler "kubegems.io/pkg/service/handlers/cluster"
	environmenthandler "kubegems.io/pkg/service/handlers/environment"
	eventhandler "kubegems.io/pkg/service/handlers/event"
	loginhandler "kubegems.io/pkg/service/handlers/login"
	logqueryhandler "kubegems.io/pkg/service/handlers/logquery"
	lokiloghandler "kubegems.io/pkg/service/handlers/lokilog"
	messagehandler "kubegems.io/pkg/service/handlers/message"
	"kubegems.io/pkg/service/handlers/metrics"
	microservice "kubegems.io/pkg/service/handlers/microservice"
	myinfohandler "kubegems.io/pkg/service/handlers/myinfo"
	noproxyhandler "kubegems.io/pkg/service/handlers/noproxy"
	projecthandler "kubegems.io/pkg/service/handlers/project"
	proxyhandler "kubegems.io/pkg/service/handlers/proxy"
	registryhandler "kubegems.io/pkg/service/handlers/registry"
	sel "kubegems.io/pkg/service/handlers/sels"
	systemhandler "kubegems.io/pkg/service/handlers/system"
	systemrolehandler "kubegems.io/pkg/service/handlers/systemrole"
	tenanthandler "kubegems.io/pkg/service/handlers/tenant"
	userhandler "kubegems.io/pkg/service/handlers/users"
	workloadreshandler "kubegems.io/pkg/service/handlers/workloadres"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/utils/prometheus/collector"
	"kubegems.io/pkg/version"
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
	basehandler := base.NewHandler(server)

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
	systemHandler := systemhandler.SystemHandler{BaseHandler: basehandler}
	systemHandler.RegistRouter(rg)

	// 用户
	userHandler := &userhandler.UserHandler{BaseHandler: basehandler}
	userHandler.RegistRouter(rg)

	// 系统角色
	systemroleHandler := &systemrolehandler.SystemRoleHandler{BaseHandler: basehandler}
	systemroleHandler.RegistRouter(rg)

	// 集群
	clusterHandler := &clusterhandler.ClusterHandler{BaseHandler: basehandler, InstallerOptions: server.GetOptions().Installer}
	clusterHandler.RegistRouter(rg)

	// 审计
	auditlogHandler := &auditloghandler.AuditLogHandler{BaseHandler: basehandler}
	auditlogHandler.RegistRouter(rg)

	// 租户
	tenantHandler := &tenanthandler.TenantHandler{BaseHandler: basehandler}
	tenantHandler.RegistRouter(rg)

	// 项目
	projectHandler := &projecthandler.ProjectHandler{BaseHandler: basehandler}
	projectHandler.RegistRouter(rg)

	// 消息
	messageHandler := &messagehandler.MessageHandler{BaseHandler: basehandler}
	messageHandler.RegistRouter(rg)

	// 消息
	approveHandler := &approveHandler.ApproveHandler{BaseHandler: basehandler}
	approveHandler.RegistRouter(rg)

	// 日志
	lokilogHandler := &lokiloghandler.LogHandler{BaseHandler: basehandler}
	lokilogHandler.RegistRouter(rg)

	// 事件
	eventHandler := &eventhandler.EventHandler{LogHandler: lokilogHandler}
	eventHandler.RegistRouter(rg)

	// 告警规则
	alertRuleHandler := &alerthandler.AlertsHandler{BaseHandler: basehandler}
	alertRuleHandler.RegistRouter(rg)

	metricsHandler := &metrics.MonitorHandler{BaseHandler: basehandler}
	metricsHandler.RegistRouter(rg)

	// 告警
	alertmgrHandler := &alerthandler.AlertmanagerConfigHandler{AlertsHandler: alertRuleHandler}
	alertmgrHandler.RegistRouter(rg)

	// 环境
	environmentHandler := &environmenthandler.EnvironmentHandler{BaseHandler: basehandler}
	environmentHandler.RegistRouter(rg)

	// 当前个人信息
	myHandler := &myinfohandler.MyHandler{BaseHandler: basehandler}
	myHandler.RegistRouter(rg)

	// 应用商店
	appstoreHandler := &appstorehandler.AppstoreHandler{BaseHandler: basehandler, AppStoreOpt: opts.Appstore}
	appstoreHandler.RegistRouter(rg)

	// 镜像仓库
	registryHandler := &registryhandler.RegistryHandler{BaseHandler: basehandler}
	registryHandler.RegistRouter(rg)

	// 日志查询历史
	logqueryhistoryHandler := &logqueryhandler.LogQueryHistoryHandler{BaseHandler: basehandler}
	logqueryhistoryHandler.RegistRouter(rg)

	// 日志查询快照
	logquerysnapshotHandler := &logqueryhandler.LogQuerySnapshotHandler{BaseHandler: basehandler}
	logquerysnapshotHandler.RegistRouter(rg)

	// 非反向代理资源[HPA]
	hpaHandler := &noproxyhandler.HpaHandler{BaseHandler: basehandler}
	hpaHandler.RegistRouter(rg)

	// 非反向代理资源[PVC]
	pvcHandler := &noproxyhandler.PersistentVolumeClaimHandler{BaseHandler: basehandler}
	pvcHandler.RegistRouter(rg)

	// 非反向代理资源[卷快照]
	volumeSnapshotHandler := &noproxyhandler.VolumeSnapshotHandler{BaseHandler: basehandler}
	volumeSnapshotHandler.RegistRouter(rg)

	//  资源统计相关
	workloadHandler := &workloadreshandler.WorkloadHandler{BaseHandler: basehandler}
	workloadHandler.RegistRouter(rg)

	// sels
	selHandler := &sel.SelsHandler{BaseHandler: basehandler}
	selHandler.RegistRouter(rg)

	// app handler
	appHandler := applicationhandler.MustNewApplicationDeployHandler(server, basehandler)
	appHandler.RegistRouter(rg)

	// microservice  handler
	microservicehandler := microservice.NewMicroServiceHandler(basehandler, server.GetOptions().Microservice)
	microservicehandler.RegistRouter(rg)

	// workload 的反向代理
	proxyHandler := proxyhandler.ProxyHandler{BaseHandler: basehandler}
	rg.Any("/proxy/cluster/:cluster/*action", proxyHandler.Proxy)
	router.Any("/v1/service-proxy/cluster/:cluster/namespace/:namespace/service/:service/port/:port/*action", proxyHandler.ProxyService)

	return nil
}
