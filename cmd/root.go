package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Set via ldflags at build time.
	version = "dev"
)

var rootCmd = &cobra.Command{
	Use:   "ap",
	Short: "Agent Protocol CLI",
	Long:  "ap is the command-line interface for Agent Protocol.",
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ap %s\n", version)
	},
}
