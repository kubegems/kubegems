package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/kubegems/pkg/model/deployment"
	"kubegems.io/kubegems/pkg/model/store"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/version"
)

func NewModelsCmd() *cobra.Command {
	options := deployment.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "models",
		Short:        "run models",
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
	cmd.AddCommand(newModelsStoreCmd())
	return cmd
}

func newModelsStoreCmd() *cobra.Command {
	storeoption := store.DefaultStoreOptions()
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
