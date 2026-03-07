package cmd

import (
	"fmt"
	"strings"

	"go.agentprotocol.cloud/cli/internal/auth"
	"go.agentprotocol.cloud/cli/internal/config"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

func authenticatedClient() (*config.Config, *auth.Token, *controlplane.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading config: %w", err)
	}
	token, err := auth.EnsureValid(cfg)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("not authenticated — run 'ap auth login' first")
	}
	client := controlplane.NewClient(cfg.ControlPlaneBaseURL, token.AccessToken)
	return cfg, token, client, nil
}

func apKeyVaultDocsURL(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.DashboardBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/docs/architecture"
}
