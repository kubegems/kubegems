package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/kubegems/pkg/msgbus"
	"kubegems.io/kubegems/pkg/msgbus/options"
	"kubegems.io/kubegems/pkg/utils/config"
	"kubegems.io/kubegems/pkg/version"
)

func NewMsgbusCmd() *cobra.Command {
	options := options.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "msgbus",
		Short:        "run msgbus",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			return msgbus.Run(ctx, options)
		},
	}
	cmd.AddCommand(genCfgCmd)
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}
