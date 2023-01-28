// Copyright 2022 The kubegems.io Authors
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

package agent

import (
	"context"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"kubegems.io/kubegems/pkg/agent/apis"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/otel"
	"kubegems.io/kubegems/pkg/utils/pprof"
	"kubegems.io/kubegems/pkg/utils/prometheus"
	"kubegems.io/kubegems/pkg/utils/prometheus/exporter"
	"kubegems.io/kubegems/pkg/utils/system"
)

type Options struct {
	DebugMode bool                        `json:"debugmode,omitempty" description:"enable debug mode"`
	LogLevel  string                      `json:"loglevel,omitempty"`
	System    *system.Options             `json:"system,omitempty"`
	API       *apis.Options               `json:"api,omitempty"`
	Kubectl   *apis.KubectlOptions        `json:"kubectl,omitempty" description:"kubectl options"`
	Exporter  *prometheus.ExporterOptions `json:"exporter,omitempty"`
	Otel      *otel.Options               `json:"otel,omitempty"`
}

func DefaultOptions() *Options {
	debugmode, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	defaultoptions := &Options{
		DebugMode: debugmode,
		LogLevel:  "debug",
		System:    system.NewDefaultOptions(),
		API:       apis.NewDefaultOptions(),
		Kubectl:   apis.NewDefaultKubectlOptions(),
		Exporter:  prometheus.DefaultExporterOptions(),
	}
	defaultoptions.System.Listen = ":8041"
	return defaultoptions
}

func Run(ctx context.Context, options *Options) error {
	log.SetLevel(options.LogLevel)
	ctx = log.NewContext(ctx, log.LogrLogger)

	if options.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	c, err := cluster.NewLocalAgentClusterAndStart(ctx)
	if err != nil {
		return err
	}

	exporterHandler := exporter.NewHandler("gems_agent", map[string]exporter.Collectorfunc{
		"plugin":                 exporter.NewPluginCollectorFunc(c), // plugin exporter
		"request":                exporter.NewRequestCollector(),     // http exporter
		"cluster_component_cert": exporter.NewCertCollectorFunc(),    // cluster component cert
	})

	if err := otel.Init(ctx, options.Otel); err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return apis.Run(ctx, c, options.System, options.API, options.Kubectl)
	})
	eg.Go(func() error {
		return pprof.Run(ctx)
	})
	eg.Go(func() error {
		return exporterHandler.Run(ctx, options.Exporter)
	})
	return eg.Wait()
}
