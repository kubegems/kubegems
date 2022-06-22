package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/kubegems/pkg/model/deployment"
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
	return cmd
}
