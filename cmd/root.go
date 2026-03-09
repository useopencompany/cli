package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/output"
)

var (
	// Set via ldflags at build time.
	version       = "dev"
	readBuildInfo = debug.ReadBuildInfo
	osArgs        = os.Args
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "ap",
	Short: "Agent Protocol CLI",
	Long:  "ap is the command-line interface for Agent Protocol.",
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

// isJSONOutput returns true when the --json flag was explicitly set by the user.
func isJSONOutput() bool {
	f := rootCmd.PersistentFlags().Lookup("json")
	return f != nil && f.Changed
}

// Execute runs the root command. When --json is active and the command errors,
// a structured JSON error is written to stderr and the process exits non-zero.
func Execute() error {
	maybeWarnIfOutdated(os.Stderr)
	err := rootCmd.Execute()
	if err != nil && isJSONOutput() {
		output.ErrorJSON(os.Stderr, err)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
	return err
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s\n", binaryName(), resolvedVersion())
	},
}

func binaryName() string {
	if len(osArgs) == 0 {
		return rootCmd.Name()
	}
	name := strings.TrimSpace(filepath.Base(osArgs[0]))
	if name == "" {
		return rootCmd.Name()
	}
	return name
}

func resolvedVersion() string {
	trimmed := strings.TrimSpace(version)
	if trimmed != "" && trimmed != "dev" {
		return trimmed
	}

	info, ok := readBuildInfo()
	if !ok || info == nil {
		return trimmed
	}
	if buildVersion := strings.TrimSpace(info.Main.Version); buildVersion != "" && buildVersion != "(devel)" {
		return buildVersion
	}
	if derived := buildInfoVersion(info); derived != "" {
		return derived
	}
	return trimmed
}

func buildInfoVersion(info *debug.BuildInfo) string {
	if info == nil {
		return ""
	}

	var revision string
	var modified bool

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = strings.TrimSpace(setting.Value)
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if revision == "" {
		return ""
	}

	short := revision
	if len(short) > 12 {
		short = short[:12]
	}
	if modified {
		return "dev+" + short + "-dirty"
	}
	return "dev+" + short
}
