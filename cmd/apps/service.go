package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	_ "kubegems.io/docs/swagger"
	"kubegems.io/pkg/models"
	"kubegems.io/pkg/service"
	"kubegems.io/pkg/service/options"
	"kubegems.io/pkg/utils/config"
	"kubegems.io/pkg/version"
)

func NewServiceCmd() *cobra.Command {
	options := options.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "service",
		Short:        "run service",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return service.Run(ctx, options)
		},
	}
	cmd.AddCommand(
		newGenServiceCfgCmd(),
		newServiceMigrateCmd(),
	)
	options.RegistFlags("", cmd.Flags())
	return cmd
}

func newGenServiceCfgCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gencfg",
		Short: "generate config template",
		Run: func(_ *cobra.Command, _ []string) {
			config.GenerateConfig(options.DefaultOptions())
		},
	}
}

func newServiceMigrateCmd() *cobra.Command {
	options := options.DefaultOptions()
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "execute migrate, init datbases and base data (use server config)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			config.Parse(cmd.Flags())
			return models.MigrateDatabaseAndInitData(options.Mysql, options.Redis)
		},
	}
	options.RegistFlags("", cmd.Flags())
	return cmd
}
