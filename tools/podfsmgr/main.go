package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "podfsmgr",
		Short: "pod file manager utilities for kubegems",
	}
	root.AddCommand(lsCmd())
	err := root.Execute()
	if err != nil {
		println(err.Error())
		os.Exit(128)
	}
}

func lsCmd() *cobra.Command {
	lscmd := &cobra.Command{
		Use:  "ls",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs, err := ListDir(args[0])
			if err != nil {
				return err
			}
			fs.Show()
			return nil
		},
	}
	return lscmd
}
