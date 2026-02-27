package main

import (
	"os"

	"go.agentprotocol.cloud/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
