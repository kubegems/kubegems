package agent

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"kubegems.io/pkg/agent/apis"
	"kubegems.io/pkg/agent/cluster"
	"kubegems.io/pkg/agent/collector"
	"kubegems.io/pkg/agent/indexer"
	"kubegems.io/pkg/log"
	"kubegems.io/pkg/utils"
	"kubegems.io/pkg/utils/exporter"
	"kubegems.io/pkg/utils/kube"
	"kubegems.io/pkg/utils/pprof"
	"kubegems.io/pkg/utils/prometheus"
	basecollector "kubegems.io/pkg/utils/prometheus/collector" // http exporter
)

type Options struct {
	Syetem    *apis.Options
	Exporter  *exporter.ExporterOptions
	DebugMode bool   `yaml:"debugmode" head_comment:"enable debug mode"`
	LogLevel  string `yaml:"loglevel"`
}

func (o *Options) RegistFlags(prefix string, fs *pflag.FlagSet) {
	fs.BoolVar(&o.DebugMode, utils.JoinFlagName(prefix, "debugmode"), o.DebugMode, "enable debud mode")
	fs.StringVar(&o.LogLevel, utils.JoinFlagName(prefix, "loglevel"), o.LogLevel, "log level")
	o.Syetem.RegistFlags("system", fs)
	o.Exporter.RegistFlags("exporter", fs)
}

func DefaultOptions() *Options {
	return &Options{
		DebugMode: false,
		LogLevel:  "debug",
		Syetem:    apis.DefaultOptions(),
		Exporter:  exporter.DefaultExporterOptions(),
	}
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
	exporterHandler := exporter.NewHandler(options.Exporter.IncludeExporterMetrics, options.Exporter.MaxRequests, log.GlobalLogger.Sugar())

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return apis.Run(ctx, c, options.Syetem)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		return prometheus.RunExporter(ctx, options.Exporter, exporterHandler)
	})
	return eg.Wait()
}
