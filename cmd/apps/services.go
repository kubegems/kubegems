package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/services"
	"kubegems.io/pkg/utils/config"
)

func NewServicesCmd() *cobra.Command {
	options := options.DefaultOptions()

	cmd := &cobra.Command{
		Use:          "services",
		Short:        "run services",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return services.Run(ctx, options)
		},
	}
	cmd.AddCommand(
		newGenServiceCfgCmd(),
		newServiceMigrateCmd(),
	)
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}
