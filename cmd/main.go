package main

import (
	"fmt"
	"os"

	"github.com/kubegems/gems/cmd/apps"
	"github.com/kubegems/gems/pkg/version"
	"github.com/spf13/cobra"
)

const ErrExitCode = 1

func main() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(ErrExitCode)
	}
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kubegems",
		Short:   "kubegems allinone binary",
		Version: version.Get().String(),
	}
	cmd.AddCommand(
		apps.NewVersionCmd(),
		apps.NewControllerCmd(),
		apps.NewAgentCmd(),
		apps.NewServiceCmd(),
		apps.NewMsgbusCmd(),
		apps.NewWorkerCmd(),
	)

	return cmd
}
