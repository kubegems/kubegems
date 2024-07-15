// Copyright 2023 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apps

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel"
	"golang.org/x/sync/errgroup"
	applicationhandler "kubegems.io/kubegems/pkg/apps/application"
	"kubegems.io/kubegems/pkg/log"
	msgbusclient "kubegems.io/kubegems/pkg/msgbus/client"
	"kubegems.io/kubegems/pkg/service/aaa"
	"kubegems.io/kubegems/pkg/service/aaa/audit"
	"kubegems.io/kubegems/pkg/service/aaa/auth"
	"kubegems.io/kubegems/pkg/service/aaa/authorization"
	appstorehandler "kubegems.io/kubegems/pkg/service/handlers/appstore"
	"kubegems.io/kubegems/pkg/service/handlers/base"
	modelscache "kubegems.io/kubegems/pkg/service/models/cache"
	"kubegems.io/kubegems/pkg/service/models/validate"
	"kubegems.io/kubegems/pkg/version"

	"kubegems.io/kubegems/pkg/utils/agents"
	"kubegems.io/kubegems/pkg/utils/argo"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/git"
	"kubegems.io/kubegems/pkg/utils/helm"
	"kubegems.io/kubegems/pkg/utils/jwt"
	"kubegems.io/kubegems/pkg/utils/msgbus"
	"kubegems.io/kubegems/pkg/utils/pprof"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/exporter"
	"kubegems.io/kubegems/pkg/utils/redis"
	"kubegems.io/kubegems/pkg/utils/system"
)

type ApplicationsOptions struct {
	System   *system.Options             `json:"system,omitempty"`
	Appstore *helm.Options               `json:"appstore,omitempty"`
	Argo     *argo.Options               `json:"argo,omitempty"`
	Exporter *prometheus.ExporterOptions `json:"exporter,omitempty"`
	Git      *git.Options                `json:"git,omitempty"`
	LogLevel string                      `json:"logLevel,omitempty"`
	Msgbus   *msgbus.Options             `json:"msgbus,omitempty"`
	JWT      *jwt.Options                `json:"jwt,omitempty"`
	Mysql    *database.Options           `json:"mysql,omitempty"`
	Redis    *redis.Options              `json:"redis,omitempty"`
}

func NewDefaultAppsOptions() *ApplicationsOptions {
	return &ApplicationsOptions{
		System:   system.NewDefaultOptions(),
		Appstore: helm.NewDefaultOptions(),
		Argo:     argo.NewDefaultArgoOptions(),
		Exporter: prometheus.DefaultExporterOptions(),
		Git:      git.NewDefaultOptions(),
		Msgbus:   msgbus.DefaultMsgbusOptions(),
		Mysql:    database.NewDefaultOptions(),
		Redis:    redis.NewDefaultOptions(),
		JWT:      jwt.DefaultOptions(),
		LogLevel: "info",
	}
}

func NewAppsCmd() *cobra.Command {
	options := NewDefaultAppsOptions()
	cmd := &cobra.Command{
		Use:     "apps",
		Version: version.Get().String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

// Run runs the application
// nolint:funlen
func Run(ctx context.Context, opts *ApplicationsOptions) error {
	log.SetLevel(opts.LogLevel)
	ctx = logr.NewContext(ctx, log.LogrLogger)
	// redis
	rediscli, err := redis.NewClient(opts.Redis)
	if err != nil {
		return fmt.Errorf("failed to init redis: %w", err)
	}
	_, err = rediscli.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}
	// database
	db, err := database.NewDatabase(opts.Mysql)
	if err != nil {
		return fmt.Errorf("failed to init database: %w", err)
	}
	// agents
	agentclientset, err := agents.NewClientSet(db)
	if err != nil {
		return fmt.Errorf("failed to init agents: %w", err)
	}
	// git
	gitprovider, err := git.NewProvider(opts.Git)
	if err != nil {
		return fmt.Errorf("failed to init git: %w", err)
	}
	// argo
	argocli, err := argo.NewClient(ctx, opts.Argo)
	if err != nil {
		log.Errorf("failed to init argo: %v", err)
		return err
	}
	// validator
	validate.InitValidator(db.DB())

	// cache
	var cache modelscache.ModelCache
	if rediscli != nil {
		cache = modelscache.NewRedisModelCache(db.DB(), rediscli)
	} else {
		cache = modelscache.NewMemoryModelCache(db.DB())
	}
	if err := cache.BuildCacheIfNotExist(); err != nil {
		return err
	}
	// user interface
	userif := aaa.NewUserInfoHandler()
	auditInstance := audit.NewAuditMiddleware(db.DB(), cache, userif)
	// 注册中间件
	tracer := otel.GetTracerProvider().Tracer("kubegems.io/kubegems")

	// app handler
	router := gin.New()
	router.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "healthy"}) })
	rg := router.Group("v1")

	for _, mw := range []func(*gin.Context){
		// authc
		auth.NewAuthMiddleware(opts.JWT, userif, tracer).FilterFunc,
		// audit
		auditInstance.Middleware(),
	} {
		rg.Use(mw)
	}
	// register router
	RegistRouter(rg, gitprovider, argocli, opts.Appstore, base.NewHandler(
		auditInstance,
		&authorization.DefaultPermissionManager{Cache: cache, Userif: userif},
		userif,
		agentclientset,
		db,
		msgbusclient.NewMessageBusClient(db, opts.Msgbus),
		cache,
	))
	// run
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return system.ListenAndServeContext(ctx, opts.System.Listen, nil, router)
	})
	eg.Go(func() error {
		// 启动prometheus exporter
		return exporter.NewHandler("gems_server", map[string]exporter.Collectorfunc{
			"application": exporter.NewApplicationCollector(argocli),
		}).Run(ctx, opts.Exporter)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	return eg.Wait()
}

func RegistRouter(rg *gin.RouterGroup, git git.Provider, argo *argo.Client, appstore *helm.Options, basehandler base.BaseHandler) error {
	appHandler := applicationhandler.MustNewApplicationDeployHandler(git, argo, basehandler)
	appHandler.RegistRouter(rg)
	appstoreHandler := &appstorehandler.AppstoreHandler{BaseHandler: basehandler, AppStoreOpt: appstore}
	appstoreHandler.RegistRouter(rg)
	return nil
}
