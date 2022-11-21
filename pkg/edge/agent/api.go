package agent

import (
	"context"

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/agent/apis"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/kube"
	"kubegems.io/kubegems/pkg/utils/system"
)

type AgentAPI struct{}

func (a *AgentAPI) Run(ctx context.Context, listen string) error {
	rest, err := kube.AutoClientConfig()
	if err != nil {
		return err
	}
	c, err := cluster.NewCluster(rest)
	if err != nil {
		return err
	}
	ginr := gin.New()
	log.SetGinDebugPrintRouteFunc(log.GlobalLogger)
	ginr.Use(
		log.DefaultGinLoggerMideare(),
		gin.Recovery(),
	)
	ginhandler, err := apis.Routes(ctx, c, apis.NewDefaultOptions(), apis.NewDefaultDebugOptions())
	if err != nil {
		return err
	}
	ginr.Any("/*path", ginhandler)
	return system.ListenAndServeContext(ctx, listen, nil, ginr)
}
