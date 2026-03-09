package spawn

import (
	"testing"

	"go.agentprotocol.cloud/cli/internal/controlplane"
)

func TestFriendlySessionError(t *testing.T) {
	apiErr := &controlplane.APIError{
		Status: 409,
		Method: "POST",
		Path:   "/v1/operator/sessions",
		Code:   "AGENT_NOT_READY",
		Msg:    "agent opencompany/executive-assistant is not ready in the active workspace: missing connections: gmail/gws",
	}
	if got := friendlySessionError(apiErr); got == nil || got.Error() != apiErr.Msg {
		t.Fatalf("friendlySessionError() = %v, want %q", got, apiErr.Msg)
	}
}
