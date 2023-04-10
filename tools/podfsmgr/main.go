// Copyright 2023 The kubegems.io Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
