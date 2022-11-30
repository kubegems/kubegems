package main

import (
	"fmt"
	"os"

	"kubegems.io/kubegems/pkg/edge"
)

const ErrExitCode = 1

func main() {
	if err := edge.NewEdgeAgentCmd().Execute(); err != nil {
		fmt.Println(err.Error())
		os.Exit(ErrExitCode)
	}
}
