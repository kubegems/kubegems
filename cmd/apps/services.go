package apps

import (
	"github.com/spf13/cobra"
	"kubegems.io/pkg/services"
)

func NewServicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "services",
		Short:        "run services",
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, _ []string) {
			services.Run()
		},
	}
	return cmd
}
