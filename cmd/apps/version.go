package apps

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"kubegems.io/pkg/version"
)

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "show version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(version.Get())
		},
	}
}
