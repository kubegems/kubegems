package apps

/*
import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/pkg/utils/config"
	"kubegems.io/pkg/utils/database"
	"kubegems.io/pkg/v2/models"
	"kubegems.io/pkg/v2/services"
	"kubegems.io/pkg/v2/services/options"
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
		newGenServicesCfgCmd(),
		newServicesMigrateCmd(),
	)
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

func newGenServicesCfgCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gencfg",
		Short: "generate config template",
		Run: func(_ *cobra.Command, _ []string) {
			config.GenerateConfig(options.DefaultOptions())
		},
	}
}

func newServicesMigrateCmd() *cobra.Command {
	options := &MigratOptions{
		Mysql:    database.NewDefaultOptions(),
		InitData: false,
	}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "execute migrate, init datbases and base data (use server config)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			config.Parse(cmd.Flags())
			return models.MigrateDatabaseAndInitData(options.Mysql, options.InitData)
		},
	}
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}
*/
