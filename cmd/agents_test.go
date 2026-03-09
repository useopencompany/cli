package cmd

import (
	"testing"

	"go.agentprotocol.cloud/cli/internal/controlplane"
)

func TestAgentInstallNextStep(t *testing.T) {
	ready := &controlplane.Agent{
		ID:               "opencompany/co-ceo",
		Version:          "1.0.0",
		InstalledVersion: "1.0.0",
	}
	if got := agentInstallNextStep(ready); got != "ap spawn --agent opencompany/co-ceo" {
		t.Fatalf("agentInstallNextStep() = %q", got)
	}

	notReady := &controlplane.Agent{
		ID:               "opencompany/executive-assistant",
		Version:          "1.0.0",
		InstalledVersion: "1.0.0",
		Readiness: controlplane.AgentReadiness{
			MissingConnections: []string{"gmail/gws"},
		},
	}
	want := "connect the missing integrations/permissions above, then run ap agents show opencompany/executive-assistant"
	if got := agentInstallNextStep(notReady); got != want {
		t.Fatalf("agentInstallNextStep() = %q, want %q", got, want)
	}
}
