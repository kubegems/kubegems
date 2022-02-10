package apps

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kubegems/gems/pkg/agent"
	"github.com/kubegems/gems/pkg/utils/config"
	"github.com/kubegems/gems/pkg/version"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
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
	options.RegistFlags("", cmd.Flags())
	return cmd
}

var genCfgCmd = &cobra.Command{
	Use:   "gencfg",
	Short: "generate config template",
	RunE: func(_ *cobra.Command, _ []string) error {
		opts := agent.DefaultOptions()
		tplout, err := yaml.Marshal(opts)
		if err != nil {
			return err
		}
		fmt.Println(string(tplout))
		return nil
	},
}
