package service

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	kialiconfig "github.com/kiali/kiali/config"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/log"
	msgbus "kubegems.io/pkg/msgbus/client"
	"kubegems.io/pkg/server/define"
	"kubegems.io/pkg/service/aaa"
	"kubegems.io/pkg/service/aaa/audit"
	auth "kubegems.io/pkg/service/aaa/authentication"
	"kubegems.io/pkg/service/aaa/authorization"
	"kubegems.io/pkg/service/models"
	"kubegems.io/pkg/service/models/validate"
	"kubegems.io/pkg/service/oauth"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/service/routers"
	"kubegems.io/pkg/utils/agents"
	"kubegems.io/pkg/utils/argo"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/git"
	_ "kubegems.io/pkg/utils/kube" // 用于 AddToSchema
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus"
	"kubegems.io/pkg/utils/prometheus/collector"
	"kubegems.io/pkg/utils/redis"
	"kubegems.io/pkg/utils/tracing"
)

type Dependencies struct {
	Options   *options.Options
	Redis     *redis.Client
	Databse   *database.Database
	Argocli   *argo.Client
	Git       *git.SimpleLocalProvider
	Agentscli *agents.ClientSet
}

func prepareDependencies(ctx context.Context, options *options.Options) (*Dependencies, error) {
	// logger
	log.SetLevel(options.LogLevel)

	// tracing
	tracing.Init(ctx)

	// redis
	rediscli, err := redis.NewClient(options.Redis)
	if err != nil {
		return nil, err
	}

	// database
	models.InitRedis(rediscli) // models 中hook需要redis
	db, err := database.NewDatabase(options.Mysql)
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
	argocli, err := argo.NewClient(ctx, options.Argo)
	if err != nil {
		return nil, err
	}
	deps := &Dependencies{
		Redis:     rediscli,
		Databse:   db,
		Argocli:   argocli,
		Git:       gitprovider,
		Agentscli: agentclientset,
	}
	return deps, nil
}

type Service struct {
	Deps Dependencies
}

func Run(ctx context.Context, options *options.Options) error {
	ctx = log.NewContext(ctx, log.LogrLogger)
	deps, err := prepareDependencies(ctx, options)
	if err != nil {
		return fmt.Errorf("failed init dependencies: %v", err)
	}

	if !options.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	// validator
	validate.InitValidator(deps.Databse.DB())

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

	// 初始化数据库中的系统配置
	models.InitConfig(deps.Databse.DB())

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
		log.FromContextOrDiscard(ctx).Info("start listen", "addr", httpserver.Addr)
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
		return auditInstance.Consumer(ctx)
	})
	return eg.Wait()
}
