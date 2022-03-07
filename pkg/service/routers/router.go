package routers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/log"
	msgbus "kubegems.io/pkg/msgbus/client"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/aaa/audit"
	auth "kubegems.io/pkg/service/aaa/authentication"
	"kubegems.io/pkg/service/aaa/authorization"
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
	microserviceoptions "kubegems.io/pkg/service/handlers/microservice/options"
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
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/service/models/validate"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/oauth"
	"kubegems.io/pkg/utils/prometheus/exporter"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/system"
	"kubegems.io/pkg/utils/tracing"
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

type Router struct {
	Opts          *options.Options
	OnlineOptions *options.OnlineOptions
	Agents        *agents.ClientSet
	Database      *database.Database
	Redis         *redis.Client
	Argo          *argo.Client

	auditInstance *audit.DefaultAuditInstance
	gin           *gin.Engine
}

func (r *Router) Run(ctx context.Context, system *system.Options) error {
	if err := r.Complete(); err != nil {
		return err
	}
	httpserver := &http.Server{
		Addr:    system.Listen,
		Handler: r.gin,
		BaseContext: func(l net.Listener) context.Context {
			return ctx // 注入basecontext
		},
	}

	// run
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		log.FromContextOrDiscard(ctx).Info("start listen", "addr", httpserver.Addr)
		go func() {
			<-ctx.Done()
			httpserver.Close()
		}()
		return httpserver.ListenAndServe()
	})
	eg.Go(func() error {
		return r.auditInstance.Consumer(ctx)
	})
	return eg.Wait()
}

func (r *Router) Complete() error {
	// validator
	validate.InitValidator(r.Database.DB())
	// oauth
	oauthtool := oauth.NewOauthTool(r.OnlineOptions.Oauth)
	// user interface
	userif := aaa.NewUserInfoHandler()
	// cache
	cache := &models.CacheLayer{DataBase: r.Database, Redis: r.Redis}
	// audit
	r.auditInstance = audit.NewAuditMiddleware(r.Database.DB(), cache, userif)

	// base handler
	basehandler := base.NewHandler(
		r.auditInstance,
		&authorization.DefaultPermissionChecker{Cache: cache, Userif: userif},
		userif,
		r.Agents,
		r.Database,
		r.Redis,
		msgbus.NewMessageBusClient(r.Database, r.Opts.Msgbus),
		cache,
	)

	// init gin
	if !r.Opts.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	r.gin = gin.New()
	router := r.gin

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))

	dir, _ := os.Getwd()
	router.StaticFS("/lokiExport", http.Dir(dir+"/lokiExport"))

	authMiddleware, err := auth.NewAuthMiddleware(r.Opts.JWT, r.Database, r.Redis, aaa.NewUserInfoHandler())
	if err != nil {
		return err
	}

	globalMiddlewares := []func(*gin.Context){
		// prometheus request metrics
		exporter.GetRequestCollector().HandlerFunc(),
		// logger
		log.DefaultGinLoggerMideare(),
		// panic recovery
		gin.Recovery(),
		// http tracing
		tracing.GinMiddleware(),
		// real ip tracking
		RealClientIPMiddleware(),
	}
	for _, middleware := range globalMiddlewares {
		router.Use(middleware)
	}

	router.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "healthy"}) })
	router.GET("/version", func(c *gin.Context) { c.JSON(http.StatusOK, version.Get()) })
	router.GET("/v1/version", func(c *gin.Context) { handlers.OK(c, version.Get()) })

	// 登录和认证相关
	oauth := loginhandler.OAuthHandler{
		Midware:   authMiddleware,
		OauthTool: oauthtool,
	}
	router.POST("/v1/login", oauth.LoginHandler)
	router.GET("/v1/oauth/addr", oauth.GetOauthAddr)
	router.GET("/v1/oauth/callback", oauth.GetOauthToken)

	rg := router.Group("v1")

	// 注册中间件
	apiMidwares := []func(*gin.Context){
		// authc
		authMiddleware.MiddlewareFunc(),
		// audit
		r.auditInstance.Middleware(),
	}
	for _, mw := range apiMidwares {
		rg.Use(mw)
	}
	// 选项
	systemHandler := systemhandler.SystemHandler{
		BaseHandler:   basehandler,
		OnlineOptions: r.OnlineOptions,
	}
	systemHandler.RegistRouter(rg)

	// 用户
	userHandler := &userhandler.UserHandler{BaseHandler: basehandler}
	userHandler.RegistRouter(rg)

	// 系统角色
	systemroleHandler := &systemrolehandler.SystemRoleHandler{BaseHandler: basehandler}
	systemroleHandler.RegistRouter(rg)

	// 集群
	clusterHandler := &clusterhandler.ClusterHandler{
		BaseHandler:      basehandler,
		InstallerOptions: r.OnlineOptions.Installer,
	}
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
	alertRuleHandler := &alerthandler.AlertsHandler{
		BaseHandler:    basehandler,
		MonitorOptions: r.OnlineOptions.Monitor,
	}
	alertRuleHandler.RegistRouter(rg)

	metricsHandler := &metrics.MonitorHandler{
		BaseHandler:   basehandler,
		OnlineOptions: r.OnlineOptions,
	}
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
	appstoreHandler := &appstorehandler.AppstoreHandler{BaseHandler: basehandler, AppStoreOpt: r.Opts.Appstore}
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
	appHandler := applicationhandler.MustNewApplicationDeployHandler(r.Opts.Git, r.Argo, basehandler)
	appHandler.RegistRouter(rg)

	// microservice  handler
	// TODO: kiali在每个集群配置可能不相同，先写死，后面看要不要支持配置
	microservicehandler := microservice.NewMicroServiceHandler(basehandler, microserviceoptions.NewDefaultOptions())
	microservicehandler.RegistRouter(rg)

	// workload 的反向代理
	proxyHandler := proxyhandler.ProxyHandler{BaseHandler: basehandler}
	rg.Any("/proxy/cluster/:cluster/*action", proxyHandler.Proxy)
	router.Any("/v1/service-proxy/cluster/:cluster/namespace/:namespace/service/:service/port/:port/*action", proxyHandler.ProxyService)

	return nil
}
