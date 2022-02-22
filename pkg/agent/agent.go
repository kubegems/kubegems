package agent

import (
	"context"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/agent/apis"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/collector"
	"kubegems.io/pkg/agent/indexer"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus"
	basecollector "kubegems.io/pkg/utils/prometheus/collector" // http exporter
	"kubegems.io/pkg/utils/system"
)

type Options struct {
	DebugMode bool                      `json:"debugmode,omitempty" description:"enable debug mode"`
	LogLevel  string                    `json:"loglevel,omitempty"`
	Syetem    *system.Options           `json:"syetem,omitempty"`
	API       *apis.Options             `json:"api,omitempty"`
	Debug     *apis.DebugOptions        `json:"debug,omitempty" description:"debug options"`
	Exporter  *exporter.ExporterOptions `json:"exporter,omitempty"`
}

func DefaultOptions() *Options {
	debugmode, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	defaultoptions := &Options{
		DebugMode: debugmode,
		LogLevel:  "debug",
		Syetem:    system.NewDefaultOptions(),
		API:       apis.NewDefaultOptions(),
		Debug:     apis.NewDefaultDebugOptions(),
		Exporter:  exporter.DefaultExporterOptions(),
	}
	defaultoptions.Syetem.Listen = ":8041"
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

	exporter.SetNamespace("gems_agent")
	exporter.RegisterCollector("plugin", true, collector.NewPluginCollectorFunc(c)) // plugin exporter
	exporter.RegisterCollector("request", true, basecollector.NewRequestCollector)  // http exporter
	exporterHandler := exporter.NewHandler()

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return apis.Run(ctx, c, options.Syetem, options.API, options.Debug)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		return prometheus.RunExporter(ctx, options.Exporter, exporterHandler)
	})
	return eg.Wait()
}
