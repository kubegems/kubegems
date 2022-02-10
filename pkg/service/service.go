package service

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	kialiconfig "github.com/kiali/kiali/config"
	"github.com/kubegems/gems/pkg/kubeclient"
	"github.com/kubegems/gems/pkg/log"
	"github.com/kubegems/gems/pkg/models"
	"github.com/kubegems/gems/pkg/models/validate"
	msgbus "github.com/kubegems/gems/pkg/msgbus/client"
	"github.com/kubegems/gems/pkg/oauth"
	"github.com/kubegems/gems/pkg/server/define"
	"github.com/kubegems/gems/pkg/service/aaa"
	"github.com/kubegems/gems/pkg/service/aaa/audit"
	auth "github.com/kubegems/gems/pkg/service/aaa/authentication"
	"github.com/kubegems/gems/pkg/service/aaa/authorization"
	"github.com/kubegems/gems/pkg/service/options"
	"github.com/kubegems/gems/pkg/service/routers"
	"github.com/kubegems/gems/pkg/utils/agents"
	"github.com/kubegems/gems/pkg/utils/argo"
	"github.com/kubegems/gems/pkg/utils/database"
	"github.com/kubegems/gems/pkg/utils/exporter"
	"github.com/kubegems/gems/pkg/utils/git"
	_ "github.com/kubegems/gems/pkg/utils/kube" // 用于 AddToSchema
	"github.com/kubegems/gems/pkg/utils/pprof"
	"github.com/kubegems/gems/pkg/utils/prometheus"
	"github.com/kubegems/gems/pkg/utils/prometheus/collector"
	"github.com/kubegems/gems/pkg/utils/redis"
	"github.com/kubegems/gems/pkg/utils/tracing"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Dependencies struct {
	Ctx       context.Context
	Options   *options.Options
	Redis     *redis.Client
	Databse   *database.Database
	Argocli   *argo.Client
	Git       *git.SimpleLocalProvider
	Agentscli *agents.ClientSet
	Logger    *zap.Logger
}

func prepareDependencies(ctx context.Context, options *options.Options) (*Dependencies, error) {
	// logger
	logger, err := log.NewZapLogger(options.LogLevel, options.DebugMode)
	if err != nil {
		return nil, err
	}
	// redis
	rediscli, err := redis.NewClient(options.Redis)
	if err != nil {
		return nil, err
	}

	// database
	models.InitRedis(rediscli) // models 中hook需要redis
	db, err := database.NewDatabase(options.Mysql, logger)
	if err != nil {
		return nil, err
	}
	// agents
	agentclientset, err := agents.NewClientSet(db, options.System)
	if err != nil {
		return nil, err
	}
	// git
	gitprovider, err := git.NewProvider(options.Git)
	if err != nil {
		return nil, err
	}
	// argo
	argocli, err := argo.NewClient(ctx, options.Argo, agentclientset)
	if err != nil {
		return nil, err
	}
	deps := &Dependencies{
		Ctx:       ctx,
		Redis:     rediscli,
		Databse:   db,
		Argocli:   argocli,
		Git:       gitprovider,
		Agentscli: agentclientset,
		Logger:    logger,
	}
	return deps, nil
}

type Service struct {
	Deps Dependencies
}

func Run(ctx context.Context, options *options.Options) error {
	deps, err := prepareDependencies(ctx, options)
	if err != nil {
		return fmt.Errorf("failed init dependencies: %v", err)
	}
	ctx = deps.Ctx
	// 初始化kubeclient实例
	models.SetKubeClient(kubeclient.Init(deps.Agentscli))

	// tracing
	tracing.Init(ctx)

	log.UpdateGlobalLogger(deps.Logger)
	if !options.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	// validator
	validate.InitValidator(deps.Databse.DB())
	// logger init context
	ctx = logr.NewContext(ctx, zapr.NewLogger(deps.Logger))
	// 测试模式需要初始化数据
	if options.TestMode {
		if err := models.MigrateDatabaseAndInitData(options.Mysql, options.Redis); err != nil {
			return err
		}
	}

	exporter.SetNamespace("gems_server")
	exporter.RegisterCollector("request", true, collector.NewRequestCollector) // http exporter
	exporterHandler := exporter.NewHandler(options.Exporter.IncludeExporterMetrics, options.Exporter.MaxRequests, log.GlobalLogger.Sugar())
	// 启动prometheus exporter

	// 初始化审计实例
	auditInstance := audit.NewAuditMiddleware(deps.Databse.DB())

	// 初始化prometheus配置
	prometheus.Init()

	// 依赖的kiali库用到，需要初始化
	// FIXME: 我们用到的配置较少，初始化时填入我们的配置，如
	// AppLabelName、InjectionLabelName、VersionLabelName、IstioIdentityDomain
	// 目前没啥问题
	kialiconfig.Set(kialiconfig.NewConfig())

	router := routers.NewRouter(options)
	server := &define.Server{
		Context:          ctx,
		Options:          options,
		Cache:            deps.Redis,
		DB:               deps.Databse,
		Git:              deps.Git,
		ArgocdClient:     deps.Argocli,
		UserInterface:    aaa.NewUserInfoHandler(),
		MessageBusClient: msgbus.NewMessageBusClient(deps.Databse, options.Msgbus),
		AgentsClientSet:  deps.Agentscli,
	}
	// 给server配置缓存层
	cacheLayer := &models.CacheLayer{DataBase: deps.Databse, Redis: deps.Redis}
	server.SetCacheLayer(cacheLayer)
	auditInstance.SetCacheLayer(server)

	// 给server配置权限层
	permChecker := &authorization.DefaultPermissionChecker{CacheLayer: server}
	server.SetPermissionChecker(permChecker)

	// 给server配置审计层
	server.SetAudit(auditInstance)

	// 初始化需要注册的中间件
	authMiddleware, err := auth.InitAuthMiddleware(options.System, deps.Databse, deps.Redis, aaa.NewUserInfoHandler())
	if err != nil {
		return err
	}
	middlewares := []func(*gin.Context){
		tracing.GinMiddleware(),
		authMiddleware.MiddlewareFunc(),
	}
	if options.System.EnableAudit {
		middlewares = append(middlewares, auditInstance.Middleware())
	}

	routers.RegistRouter(router, options, server, middlewares...)
	oauth.InitOauth(options.Oauth)

	// run
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		httpserver := &http.Server{
			Addr:    options.System.Listen,
			Handler: router,
			BaseContext: func(l net.Listener) context.Context {
				return ctx // 注入basecontext
			},
		}
		logr.FromContextOrDiscard(ctx).Info("start listen", "addr", httpserver.Addr)
		go func() {
			<-ctx.Done()
			httpserver.Close()
		}()
		return httpserver.ListenAndServe()
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		return prometheus.RunExporter(ctx, options.Exporter, exporterHandler)
	})
	eg.Go(func() error {
		return auditInstance.Consumer()
	})
	return eg.Wait()
}
