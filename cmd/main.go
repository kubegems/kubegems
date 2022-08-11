// Copyright 2022 The kubegems.io Authors
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

// @title       kubegems
// @version     1.0
// @description kubegems apis swagger doc

// @BasePath /

// @securityDefinitions.apikey JWT
// @in                         header
// @name                       Authorization

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"kubegems.io/kubegems/cmd/apps"
	"kubegems.io/kubegems/pkg/version"
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
		// apps.NewServicesCmd(),
		apps.NewInstallerCmd(),
		apps.NewPluginCmd(),
		apps.NewModelsCmd(),
	)
	return cmd
}
