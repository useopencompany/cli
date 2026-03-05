package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

var integrationsCmd = &cobra.Command{
	Use:   "integrations",
	Short: "Manage integration connections",
}

var (
	connectIntegration   string
	connectProvider      string
	connectScope         string
	connectWorkspaceID   string
	connectCredentialRef string
	connectCredentials   []string
	connectToken         string
)

var integrationsConnectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Create an integration connection",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		if strings.TrimSpace(connectIntegration) == "" {
			return fmt.Errorf("--integration is required")
		}
		integration := strings.TrimSpace(connectIntegration)
		provider := strings.TrimSpace(connectProvider)
		if provider == "" {
			provider = defaultProviderForIntegration(integration)
		}
		if provider == "" {
			return fmt.Errorf("--provider is required")
		}

		credentials, err := parseCredentials(connectCredentials)
		if err != nil {
			return err
		}
		if strings.TrimSpace(connectToken) != "" {
			key := defaultTokenKey(provider, integration)
			credentials[key] = strings.TrimSpace(connectToken)
		}
		if len(credentials) == 0 {
			return fmt.Errorf("at least one credential is required (--credential KEY=VALUE or --token)")
		}

		req := controlplane.CreateIntegrationConnectionRequest{
			Integration:   integration,
			Provider:      provider,
			Scope:         strings.TrimSpace(connectScope),
			WorkspaceID:   strings.TrimSpace(connectWorkspaceID),
			CredentialRef: strings.TrimSpace(connectCredentialRef),
			Credentials:   credentials,
		}
		conn, err := client.CreateIntegrationConnection(cmd.Context(), req)
		if err != nil {
			return err
		}
		fmt.Printf("Connected %s/%s: %s\n", conn.Integration, conn.Provider, conn.ID)
		return nil
	},
}

var (
	listIntegration string
	listProvider    string
	listRevoked     bool
)

var integrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List visible integration connections",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		rows, err := client.ListIntegrationConnections(cmd.Context(), listIntegration, listProvider, listRevoked)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			fmt.Println("No integration connections found.")
			return nil
		}
		fmt.Printf("%-44s  %-18s  %-10s  %-24s  %-8s\n", "ID", "INTEGRATION", "SCOPE", "WORKSPACE", "STATUS")
		for _, row := range rows {
			workspace := row.WorkspaceID
			if workspace == "" {
				workspace = "-"
			}
			fmt.Printf("%-44s  %-18s  %-10s  %-24s  %-8s\n", row.ID, row.Integration+"/"+row.Provider, row.Scope, workspace, row.Status)
		}
		return nil
	},
}

var integrationsRevokeCmd = &cobra.Command{
	Use:   "revoke <connection-id>",
	Short: "Revoke an integration connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		status := "revoked"
		_, err = client.UpdateIntegrationConnection(cmd.Context(), args[0], controlplane.UpdateIntegrationConnectionRequest{
			Status: &status,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Revoked connection %s\n", args[0])
		return nil
	},
}

func defaultProviderForIntegration(integration string) string {
	switch strings.TrimSpace(strings.ToLower(integration)) {
	case "linear":
		return "linear"
	case "slack":
		return "slack"
	case "google-workspace", "gmail", "google-calendar", "google-drive":
		return "gws"
	default:
		return ""
	}
}

func defaultTokenKey(provider, integration string) string {
	p := strings.TrimSpace(strings.ToLower(provider))
	switch p {
	case "linear":
		return "LINEAR_API_KEY"
	case "slack":
		return "SLACK_BOT_TOKEN"
	case "gws":
		return "GOOGLE_ACCESS_TOKEN"
	}
	switch strings.TrimSpace(strings.ToLower(integration)) {
	case "linear":
		return "LINEAR_API_KEY"
	case "slack":
		return "SLACK_BOT_TOKEN"
	default:
		return "GOOGLE_ACCESS_TOKEN"
	}
}

func parseCredentials(values []string) (map[string]string, error) {
	out := map[string]string{}
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid credential %q, expected KEY=VALUE", item)
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if key == "" || val == "" {
			return nil, fmt.Errorf("invalid credential %q, expected KEY=VALUE", item)
		}
		out[key] = val
	}
	return out, nil
}

func init() {
	integrationsConnectCmd.Flags().StringVar(&connectIntegration, "integration", "", "Integration key (linear|slack|google-workspace|gmail|google-calendar|google-drive)")
	integrationsConnectCmd.Flags().StringVar(&connectProvider, "provider", "", "Provider key (linear|slack|gws)")
	integrationsConnectCmd.Flags().StringVar(&connectScope, "scope", "user_private_workspace", "Scope (org_shared|workspace_shared|user_private_workspace)")
	integrationsConnectCmd.Flags().StringVar(&connectWorkspaceID, "workspace-id", "", "Workspace ID for workspace/user-private scopes")
	integrationsConnectCmd.Flags().StringVar(&connectCredentialRef, "credential-ref", "", "External credential reference")
	integrationsConnectCmd.Flags().StringArrayVar(&connectCredentials, "credential", nil, "Credential KEY=VALUE (repeatable)")
	integrationsConnectCmd.Flags().StringVar(&connectToken, "token", "", "Convenience token value mapped to provider-specific credential key")

	integrationsListCmd.Flags().StringVar(&listIntegration, "integration", "", "Filter by integration")
	integrationsListCmd.Flags().StringVar(&listProvider, "provider", "", "Filter by provider")
	integrationsListCmd.Flags().BoolVar(&listRevoked, "include-revoked", false, "Include revoked connections (admin only)")

	integrationsCmd.AddCommand(integrationsConnectCmd)
	integrationsCmd.AddCommand(integrationsListCmd)
	integrationsCmd.AddCommand(integrationsRevokeCmd)
	rootCmd.AddCommand(integrationsCmd)
}
