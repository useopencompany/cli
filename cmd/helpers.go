package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
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

func promptForName(label string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s: ", strings.TrimSpace(label))
		value, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		name := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
		if name != "" {
			return name, nil
		}
		fmt.Println("Name is required.")
	}
}

func fetchNamedOrgInfo(ctx context.Context, client *controlplane.Client) (*controlplane.OrgInfo, error) {
	if client == nil {
		return nil, fmt.Errorf("control plane client is required")
	}
	return client.GetOrg(ctx)
}

func namedContextLabel(info *controlplane.OrgInfo, fallback string) string {
	if info == nil {
		return fallback
	}
	orgName := strings.TrimSpace(info.OrgName)
	workspaceName := strings.TrimSpace(info.ActiveWorkspace.Name)
	switch {
	case orgName != "" && workspaceName != "" && !strings.EqualFold(orgName, workspaceName):
		return orgName + " / " + workspaceName
	case orgName != "":
		return orgName
	case workspaceName != "":
		return workspaceName
	default:
		return fallback
	}
}
