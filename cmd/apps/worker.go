package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/version"
	"kubegems.io/kubegems/pkg/worker"
)

func NewWorkerCmd() *cobra.Command {
	options := worker.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "worker",
		Short:        "run worker",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return worker.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}
