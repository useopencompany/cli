package cmd

import (
	"testing"

	"go.agentprotocol.cloud/cli/internal/controlplane"
)

func TestConnectWorkspaceIDForScopeUsesExplicitWorkspace(t *testing.T) {
	got, err := connectWorkspaceIDForScope("user_private_workspace", " ws_explicit ", &controlplane.Workspace{ID: "ws_active"})
	if err != nil {
		t.Fatalf("connectWorkspaceIDForScope() error = %v, want nil", err)
	}
	if got != "ws_explicit" {
		t.Fatalf("connectWorkspaceIDForScope() = %q, want %q", got, "ws_explicit")
	}
}

func TestConnectWorkspaceIDForScopeUsesActiveWorkspaceForWorkspaceScopes(t *testing.T) {
	got, err := connectWorkspaceIDForScope("workspace_shared", "", &controlplane.Workspace{ID: "ws_active"})
	if err != nil {
		t.Fatalf("connectWorkspaceIDForScope() error = %v, want nil", err)
	}
	if got != "ws_active" {
		t.Fatalf("connectWorkspaceIDForScope() = %q, want %q", got, "ws_active")
	}
}

func TestConnectWorkspaceIDForScopeLeavesOrgSharedEmpty(t *testing.T) {
	got, err := connectWorkspaceIDForScope("org_shared", "", nil)
	if err != nil {
		t.Fatalf("connectWorkspaceIDForScope() error = %v, want nil", err)
	}
	if got != "" {
		t.Fatalf("connectWorkspaceIDForScope() = %q, want empty string", got)
	}
}

func TestConnectWorkspaceIDForScopeErrorsWithoutActiveWorkspace(t *testing.T) {
	_, err := connectWorkspaceIDForScope("user_private_workspace", "", nil)
	if err == nil {
		t.Fatal("connectWorkspaceIDForScope() error = nil, want error")
	}
}
