package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubegems/gems/pkg/controller"
	"github.com/spf13/cobra"
)

func NewControllerCmd() *cobra.Command {
	options := controller.NewDefaultOptions()
	cmd := &cobra.Command{
		Use:   "controller",
		Short: "kubegems controller",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return controller.Run(ctx, options)
		},
	}

	cmd.Flags().StringVar(&options.MetricsAddr, "metrics-addr", options.MetricsAddr, "The address the metric endpoint binds to.")
	cmd.Flags().BoolVar(&options.EnableLeaderElection, "enable-leader-election", options.EnableLeaderElection,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	return cmd
}
