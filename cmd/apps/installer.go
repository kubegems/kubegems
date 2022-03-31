package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/pkg/installer"
	"kubegems.io/pkg/utils/config"
)

func NewInstallerCmd() *cobra.Command {
	options := installer.NewDefaultOptions()
	cmd := &cobra.Command{
		Use:   "installer",
		Short: "run installer",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return installer.Run(ctx, options)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

 