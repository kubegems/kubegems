package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	_ "kubegems.io/kubegems/docs/swagger"
	"kubegems.io/kubegems/pkg/service"
	"kubegems.io/kubegems/pkg/service/models"
	"kubegems.io/kubegems/pkg/service/options"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/utils/database"
	"kubegems.io/kubegems/pkg/utils/debug"
	"kubegems.io/kubegems/pkg/version"
)

func NewServiceCmd() *cobra.Command {
	options := options.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "service",
		Short:        "run service",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			if err := debug.ApplyPortForwardingOptions(ctx, options); err != nil {
				return err
			}
			return service.Run(ctx, options)
		},
	}
	cmd.AddCommand(
		newGenServiceCfgCmd(),
		newServiceMigrateCmd(),
	)
	config.AutoRegisterFlags(cmd.Flags(), "", options)
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

type MigratOptions struct {
	Mysql    *database.Options `json:"mysql,omitempty"`
	InitData bool              `json:"initData,omitempty" description:"insert init data into database"`
}

func newServiceMigrateCmd() *cobra.Command {
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
