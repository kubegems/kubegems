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

package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/kubegems/pkg/model/deployment"
	"kubegems.io/kubegems/pkg/model/registry"
	"kubegems.io/kubegems/pkg/model/store"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/version"
)

func NewModelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "models commands",
	}
	cmd.AddCommand(newModelsControllerCmd())
	cmd.AddCommand(newModelsStoreCmd())
	cmd.AddCommand(newModelRegistryCmd())
	return cmd
}

func newModelsControllerCmd() *cobra.Command {
	options := deployment.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "controller",
		Short:        "run controller",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return deployment.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

func newModelsStoreCmd() *cobra.Command {
	storeoption := store.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "store",
		Short:        "run store",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return store.Run(ctx, storeoption)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", storeoption)
	return cmd
}

func newModelRegistryCmd() *cobra.Command {
	options := registry.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "registry",
		Short:        "run model registry",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return registry.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}
