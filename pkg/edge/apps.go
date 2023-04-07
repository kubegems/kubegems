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
	"kubegems.io/kubegems/pkg/edge/server"
	"kubegems.io/kubegems/pkg/edge/task"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/version"
)

func NewEdgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "edge",
	}
	cmd.AddCommand(
		NewEdgeAgentCmd(),
		NewEdgeHubCmd(),
		NewEdgeServerCmd(),
		NewEdgeTaskCmd(),
	)
	return cmd
}

func NewEdgeHubCmd() *cobra.Command {
	options := hub.NewDefaultOptions()
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

func NewEdgeServerCmd() *cobra.Command {
	options := server.NewDefaultOptions()
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

func NewEdgeAgentCmd() *cobra.Command {
	options := agent.NewDefaultOptions()
	cmd := &cobra.Command{
		Use: "agent",
		Example: `
	To use SN as kubegems-edge device id from a manufacturefile:
	$ cat /etc/some-file
	SN=sn-123456
	...
	$ kubegems-edge-agent --manufacturefile=/etc/some-file --deviceidkey=SN

	Or set kubegems-edge device id from flag:
	$ kubegems-edge-agent --deviceid=sn-123456
		`,
		Version:            version.Get().String(),
		DisableFlagParsing: true, // avoid parse twice on slice args
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			if b, _ := cmd.Flags().GetBool("help"); b {
				return cmd.Help()
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return agent.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

func NewEdgeTaskCmd() *cobra.Command {
	options := task.NewDefaultOptions()
	cmd := &cobra.Command{
		Use: "task",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return task.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}
