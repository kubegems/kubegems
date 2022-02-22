package apps

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"kubegems.io/pkg/agent"
	"kubegems.io/pkg/utils/config"
	"kubegems.io/pkg/version"
)

func NewAgentCmd() *cobra.Command {
	options := agent.DefaultOptions()
	cmd := &cobra.Command{
		Use:          "agent",
		Short:        "run agent",
		SilenceUsage: true,
		Version:      version.Get().String(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := config.Parse(cmd.Flags()); err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()
			return agent.Run(ctx, options)
		},
	}
	cmd.AddCommand(genCfgCmd)
	config.AutoRegisterFlags(cmd.Flags(), "", options)
	return cmd
}

var genCfgCmd = &cobra.Command{
	Use:   "gencfg",
	Short: "generate config template",
	RunE: func(_ *cobra.Command, _ []string) error {
		opts := agent.DefaultOptions()
		config.GenerateConfig(opts)
		return nil
	},
}
