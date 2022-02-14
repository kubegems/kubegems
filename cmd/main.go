package main

// @title kubegems
// @version 1.0
// @description kubegems apis swagger doc

// @BasePath /

// @securityDefinitions.apikey JWT
// @in header
// @name Authorization

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"kubegems.io/cmd/apps"
	"kubegems.io/pkg/version"
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
