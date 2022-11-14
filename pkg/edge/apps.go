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

package edge

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/kubegems/pkg/edge/agent"
	"kubegems.io/kubegems/pkg/edge/hub"
	"kubegems.io/kubegems/pkg/edge/options"
	"kubegems.io/kubegems/pkg/edge/server"
	"kubegems.io/kubegems/pkg/utils/config"
)

/*
edge-node  --> edge-hub --> edge-server

peer1 -> peer -> peer
peer2 -> peer

func connect()
func dial()

节点平行：
- 提供本节点能够被代理访问的 IP
- 提供本节点标识符号
- 提供连接到本节点的实例
- 能够上联至上级节点，提供本节点host的节点
- 如果dial 本节点则使用本节点进行dial
*/

func NewEdgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "edge",
	}
	cmd.AddCommand(
		newEdgeAgentCmd(),
		newEdgeHubCmd(),
		newEdgeServerCmd(),
	)
	return cmd
}

func newEdgeHubCmd() *cobra.Command {
	options := options.NewDefaultHub()
	cmd := &cobra.Command{
		Use: "hub",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return hub.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

func newEdgeServerCmd() *cobra.Command {
	options := options.NewDefaultServer()
	cmd := &cobra.Command{
		Use: "server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return server.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

func newEdgeAgentCmd() *cobra.Command {
	options := options.NewDefaultAgentOptions()
	cmd := &cobra.Command{
		Use: "agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return agent.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}
