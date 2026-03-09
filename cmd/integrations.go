package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
	"golang.org/x/term"
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
	connectTokenStdin    bool
	connectCredentialsIn bool
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
		if connectTokenStdin && connectCredentialsIn {
			return fmt.Errorf("--token-stdin and --credentials-stdin cannot be combined")
		}

		integration := strings.TrimSpace(connectIntegration)
		provider := strings.TrimSpace(connectProvider)
		if provider == "" {
			provider = defaultProviderForIntegration(integration)
		}
		if provider == "" {
			return fmt.Errorf("--provider is required")
		}

		var credentials map[string]string
		switch {
		case connectCredentialsIn:
			credentials, err = parseCredentialsReader(os.Stdin)
		case connectTokenStdin:
			token, readErr := readSingleSecret(os.Stdin)
			err = readErr
			if err == nil {
				credentials = map[string]string{defaultTokenKey(provider, integration): token}
			}
		default:
			key := defaultTokenKey(provider, integration)
			token, readErr := promptSecret(cmd, "Enter "+key)
			err = readErr
			if err == nil {
				credentials = map[string]string{key: token}
			}
		}
		if err != nil {
			return err
		}
		if len(credentials) == 0 {
			return fmt.Errorf("credentials are required")
		}
		workspaceID, err := resolveConnectWorkspaceID(cmd.Context(), client, connectScope, connectWorkspaceID)
		if err != nil {
			return err
		}

		conn, err := client.CreateIntegrationConnection(cmd.Context(), controlplane.CreateIntegrationConnectionRequest{
			Integration: integration,
			Provider:    provider,
			Scope:       strings.TrimSpace(connectScope),
			WorkspaceID: workspaceID,
			Credentials: credentials,
		})
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

func parseCredentialsReader(r io.Reader) (map[string]string, error) {
	out := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		item := strings.TrimSpace(scanner.Text())
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
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func readSingleSecret(r io.Reader) (string, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(string(raw))
	if secret == "" {
		return "", fmt.Errorf("stdin did not contain a secret")
	}
	return secret, nil
}

func promptSecret(cmd *cobra.Command, prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("stdin is not a terminal; use --token-stdin or --credentials-stdin")
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s: ", prompt)
	raw, err := term.ReadPassword(fd)
	fmt.Fprintln(cmd.ErrOrStderr())
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(string(raw))
	if secret == "" {
		return "", fmt.Errorf("secret is required")
	}
	return secret, nil
}

func resolveConnectWorkspaceID(ctx context.Context, client *controlplane.Client, scope, workspaceID string) (string, error) {
	trimmedID := strings.TrimSpace(workspaceID)
	if trimmedID != "" || !connectScopeRequiresWorkspace(scope) {
		return trimmedID, nil
	}
	if client == nil {
		return "", fmt.Errorf("control plane client is required")
	}
	info, err := client.GetOrg(ctx)
	if err != nil {
		return "", err
	}
	return connectWorkspaceIDForScope(scope, "", &info.ActiveWorkspace)
}

func connectScopeRequiresWorkspace(scope string) bool {
	switch strings.TrimSpace(scope) {
	case "workspace_shared", "user_private_workspace":
		return true
	default:
		return false
	}
}

func connectWorkspaceIDForScope(scope, workspaceID string, active *controlplane.Workspace) (string, error) {
	trimmedID := strings.TrimSpace(workspaceID)
	if trimmedID != "" || !connectScopeRequiresWorkspace(scope) {
		return trimmedID, nil
	}
	if active == nil || strings.TrimSpace(active.ID) == "" {
		return "", fmt.Errorf("active workspace could not be determined; pass --workspace-id explicitly")
	}
	return strings.TrimSpace(active.ID), nil
}

func init() {
	integrationsConnectCmd.Flags().StringVar(&connectIntegration, "integration", "", "Integration key (linear|slack|google-workspace|gmail|google-calendar|google-drive)")
	integrationsConnectCmd.Flags().StringVar(&connectProvider, "provider", "", "Provider key (linear|slack|gws)")
	integrationsConnectCmd.Flags().StringVar(&connectScope, "scope", "user_private_workspace", "Scope (org_shared|workspace_shared|user_private_workspace)")
	integrationsConnectCmd.Flags().StringVar(&connectWorkspaceID, "workspace-id", "", "Workspace ID for workspace/user-private scopes (defaults to active workspace)")
	integrationsConnectCmd.Flags().BoolVar(&connectTokenStdin, "token-stdin", false, "Read a single provider token from stdin")
	integrationsConnectCmd.Flags().BoolVar(&connectCredentialsIn, "credentials-stdin", false, "Read KEY=VALUE credential lines from stdin")

	integrationsListCmd.Flags().StringVar(&listIntegration, "integration", "", "Filter by integration")
	integrationsListCmd.Flags().StringVar(&listProvider, "provider", "", "Filter by provider")
	integrationsListCmd.Flags().BoolVar(&listRevoked, "include-revoked", false, "Include revoked connections (admin only)")

	integrationsCmd.AddCommand(integrationsConnectCmd)
	integrationsCmd.AddCommand(integrationsListCmd)
	integrationsCmd.AddCommand(integrationsRevokeCmd)
	rootCmd.AddCommand(integrationsCmd)
}
