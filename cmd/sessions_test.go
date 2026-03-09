package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

func TestValidateSessionArgsRejectsUnknownSessionCommand(t *testing.T) {
	cmd := newSessionTestCommand()

	err := validateSessionArgs(cmd, []string{"start"})
	if err == nil {
		t.Fatal("validateSessionArgs() error = nil, want error")
	}
	message := err.Error()
	for _, want := range []string{
		`unknown command "start" for "ap session"`,
		"Did you mean one of these?",
		"ap spawn",
		"ap sessions",
		"ap session <ID>",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("validateSessionArgs() error = %q, want substring %q", message, want)
		}
	}
}

func TestValidateSessionArgsAllowsSessionID(t *testing.T) {
	cmd := newSessionTestCommand()

	if err := validateSessionArgs(cmd, []string{"sess_1234"}); err != nil {
		t.Fatalf("validateSessionArgs() error = %v, want nil", err)
	}
}

func TestSessionLookupErrorMapsNotFoundToFriendlyMessage(t *testing.T) {
	cmd := newSessionTestCommand()

	err := sessionLookupError(cmd, "sess_missing", &controlplane.APIError{Status: 404})
	if err == nil {
		t.Fatal("sessionLookupError() error = nil, want error")
	}
	message := err.Error()
	for _, want := range []string{
		`sess_missing`,
		"not found",
		"Run 'ap sessions' to list available sessions",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("sessionLookupError() error = %q, want substring %q", message, want)
		}
	}
	if strings.Contains(message, "control-plane") {
		t.Fatalf("sessionLookupError() error = %q, do not want raw control-plane message", message)
	}
}

func TestSessionLookupErrorPreservesNonNotFoundErrors(t *testing.T) {
	cmd := newSessionTestCommand()
	original := errors.New("boom")

	err := sessionLookupError(cmd, "sess_1234", original)
	if !errors.Is(err, original) {
		t.Fatalf("sessionLookupError() error = %v, want %v", err, original)
	}
}

func newSessionTestCommand() *cobra.Command {
	root := &cobra.Command{Use: "ap"}
	cmd := &cobra.Command{Use: "session <ID>"}
	root.AddCommand(cmd)
	return cmd
}
