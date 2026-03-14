package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolvedVersionPrefersBuildVersion(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	originalArgs := osArgs
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
		osArgs = originalArgs
	})

	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "go.agentprotocol.cloud/cli",
				Version: "v0.2.2-0.20260306085414-c52142752b03",
			},
		}, true
	}
	osArgs = []string{"/tmp/ap-dev"}

	if got, want := resolvedVersion(), "v0.2.2-0.20260306085414-c52142752b03"; got != want {
		t.Fatalf("resolvedVersion() = %q, want %q", got, want)
	}
	if got, want := binaryName(), "ap-dev"; got != want {
		t.Fatalf("binaryName() = %q, want %q", got, want)
	}
}

func TestResolvedVersionFallsBackToVCSRevision(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "c52142752b0354abb092af28c5f04124fff47444"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true
	}

	if got, want := resolvedVersion(), "dev+c52142752b03-dirty"; got != want {
		t.Fatalf("resolvedVersion() = %q, want %q", got, want)
	}
}

func TestJSONFlagRegistered(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("json")
	if f == nil {
		t.Fatal("expected --json persistent flag to be registered on rootCmd")
	}
	if f.DefValue != "false" {
		t.Fatalf("expected --json default to be \"false\", got %q", f.DefValue)
	}
}

func TestIsJSONOutputDefaultFalse(t *testing.T) {
	if isJSONOutput() {
		t.Fatal("expected isJSONOutput() to return false by default")
	}
}

func TestSilenceErrorsSet(t *testing.T) {
	if !rootCmd.SilenceErrors {
		t.Fatal("expected rootCmd.SilenceErrors to be true")
	}
}

func TestSilenceUsageSet(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Fatal("expected rootCmd.SilenceUsage to be true")
	}
}

func TestDoCmdBodyFlagExists(t *testing.T) {
	f := doCmd.Flags().Lookup("body")
	if f == nil {
		t.Fatal("expected --body flag to be registered on doCmd")
	}
}

func TestDoCmdNoLocalJSONFlag(t *testing.T) {
	// doCmd should NOT have a local --json flag; the persistent --json from root is separate
	f := doCmd.Flags().Lookup("json")
	if f != nil {
		t.Fatal("expected doCmd to NOT have a local --json flag (should use --body instead)")
	}
}

func TestResolvedVersionPrefersInjectedVersion(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	version = "v0.2.1-4-gc521427"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v0.0.0"},
		}, true
	}

	if got, want := resolvedVersion(), "v0.2.1-4-gc521427"; got != want {
		t.Fatalf("resolvedVersion() = %q, want %q", got, want)
	}
}
