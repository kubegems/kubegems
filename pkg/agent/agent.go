package agent

import (
	"context"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/agent/apis"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/indexer"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus"
	"kubegems.io/pkg/utils/prometheus/exporter"
	"kubegems.io/pkg/utils/system"
)

type Options struct {
	DebugMode bool                        `json:"debugmode,omitempty" description:"enable debug mode"`
	LogLevel  string                      `json:"loglevel,omitempty"`
	System    *system.Options             `json:"system,omitempty"`
	API       *apis.Options               `json:"api,omitempty"`
	Debug     *apis.DebugOptions          `json:"debug,omitempty" description:"debug options"`
	Exporter  *prometheus.ExporterOptions `json:"exporter,omitempty"`
}

func DefaultOptions() *Options {
	debugmode, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	defaultoptions := &Options{
		DebugMode: debugmode,
		LogLevel:  "debug",
		System:    system.NewDefaultOptions(),
		API:       apis.NewDefaultOptions(),
		Debug:     apis.NewDefaultDebugOptions(),
		Exporter:  prometheus.DefaultExporterOptions(),
	}
	defaultoptions.System.Listen = ":8041"
	return defaultoptions
}

func Run(ctx context.Context, options *Options) error {
	log.SetLevel(options.LogLevel)

	if options.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	rest, err := kube.AutoClientConfig()
	if err != nil {
		return err
	}

	c, err := cluster.NewCluster(rest)
	if err != nil {
		return err
	}

	if err := indexer.CustomIndexPods(c.GetCache()); err != nil {
		return err
	}

	go c.Start(ctx)
	c.GetCache().WaitForCacheSync(ctx)

	exporterHandler := exporter.NewHandler("gems_agent", map[string]exporter.Collectorfunc{
		"plugin":  exporter.NewPluginCollectorFunc(c), // plugin exporter
		"request": exporter.NewRequestCollector(),     // http exporter
	})

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return apis.Run(ctx, c, options.System, options.API, options.Debug)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		return exporterHandler.Run(ctx, options.Exporter)
	})
	return eg.Wait()
}
