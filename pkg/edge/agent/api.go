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

	"github.com/gin-gonic/gin"
	"kubegems.io/kubegems/pkg/agent/apis"
	"kubegems.io/kubegems/pkg/agent/cluster"
	"kubegems.io/kubegems/pkg/log"
	"kubegems.io/kubegems/pkg/utils/system"
)

type AgentAPI struct {
	cluster *cluster.Cluster
}

func (a *AgentAPI) Run(ctx context.Context, listen string) error {
	ginr := gin.New()
	gin.SetMode(gin.ReleaseMode)
	log.SetGinDebugPrintRouteFunc(log.GlobalLogger)
	ginr.Use(
		log.DefaultGinLoggerMideare(),
		gin.Recovery(),
	)
	ginhandler, err := apis.Routes(ctx, a.cluster, apis.NewDefaultOptions(), apis.NewDefaultDebugOptions())
	if err != nil {
		return err
	}
	ginr.Any("/*path", ginhandler)
	return system.ListenAndServeContext(ctx, listen, nil, ginr)
}
