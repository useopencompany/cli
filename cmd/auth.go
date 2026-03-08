package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/auth"
	"go.agentprotocol.cloud/cli/internal/config"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Agent Protocol",
	Long:  "Manage authentication. Use 'ap auth login' to sign in via WorkOS AuthKit.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to Agent Protocol",
	Long:  "Opens your browser to authenticate via WorkOS AuthKit using the device authorization flow.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		token, err := auth.DeviceFlow(cmd.Context(), cfg.WorkOSClientID)
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		token, err = ensureNamedPersonalWorkspace(cmd.Context(), cfg, token)
		if err != nil {
			return err
		}

		if err := auth.SaveToken(token); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}

		fmt.Println("\n✓ Authenticated successfully.")

		info := currentOrgInfo(cmd.Context(), cfg, token)
		if displayName := strings.TrimSpace(info.UserDisplayName); displayName != "" {
			fmt.Printf("  User:      %s\n", displayName)
		} else if userID := token.UserID(); userID != "" {
			fmt.Printf("  User:      %s\n", userID)
		}
		if orgName := strings.TrimSpace(info.OrgName); orgName != "" {
			fmt.Printf("  Org:       %s\n", orgName)
		}
		if workspaceName := strings.TrimSpace(info.ActiveWorkspace.Name); workspaceName != "" {
			fmt.Printf("  Workspace: %s\n", workspaceName)
		}

		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out of Agent Protocol",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.ClearToken(); err != nil {
			return fmt.Errorf("clearing credentials: %w", err)
		}

		fmt.Println("Logged out.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		token, err := auth.EnsureValid(cfg)
		if err != nil {
			fmt.Println("Not authenticated. Run 'ap auth login' to sign in.")
			return nil
		}

		info := currentOrgInfo(cmd.Context(), cfg, token)

		fmt.Println("✓ Authenticated")
		if displayName := strings.TrimSpace(info.UserDisplayName); displayName != "" {
			fmt.Printf("  User:      %s\n", displayName)
		} else if userID := token.UserID(); userID != "" {
			fmt.Printf("  User:      %s\n", userID)
		}
		if orgName := strings.TrimSpace(info.OrgName); orgName != "" {
			fmt.Printf("  Org:       %s\n", orgName)
		}
		if workspaceName := strings.TrimSpace(info.ActiveWorkspace.Name); workspaceName != "" {
			fmt.Printf("  Workspace: %s\n", workspaceName)
		}
		fmt.Printf("  Expires:   %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func ensureNamedPersonalWorkspace(ctx context.Context, cfg *config.Config, token *auth.Token) (*auth.Token, error) {
	if token == nil {
		return nil, fmt.Errorf("missing token")
	}
	if token.OrganizationID() != "" {
		info := currentOrgInfo(ctx, cfg, token)
		if strings.TrimSpace(info.UserDisplayName) == "" && shouldPromptForDisplayName(info.OrgName, info.OrgID) {
			displayName, err := promptForName("What should we call your personal Workspace")
			if err != nil {
				return nil, fmt.Errorf("reading display name: %w", err)
			}

			client := controlplane.NewClient(cfg.ControlPlaneBaseURL, token.AccessToken)
			bootstrap, err := client.Bootstrap(ctx, controlplane.BootstrapRequest{DisplayName: displayName})
			if err != nil {
				return nil, fmt.Errorf("updating personal Workspace: %w", err)
			}
			if bootstrap.OrganizationID != "" && bootstrap.OrganizationID != token.OrganizationID() {
				refreshed, err := auth.RefreshAccessToken(cfg.WorkOSClientID, token.RefreshToken, bootstrap.OrganizationID)
				if err != nil {
					return nil, fmt.Errorf("switching into %s: %w", bootstrap.OrganizationName, err)
				}
				return refreshed, nil
			}
		}
		return token, nil
	}

	displayName, err := promptForName("What should we call your personal Workspace")
	if err != nil {
		return nil, fmt.Errorf("reading display name: %w", err)
	}

	client := controlplane.NewClient(cfg.ControlPlaneBaseURL, token.AccessToken)
	bootstrap, err := client.Bootstrap(ctx, controlplane.BootstrapRequest{DisplayName: displayName})
	if err != nil {
		return nil, fmt.Errorf("creating personal Workspace: %w", err)
	}

	refreshed, err := auth.RefreshAccessToken(cfg.WorkOSClientID, token.RefreshToken, bootstrap.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("switching into %s: %w", bootstrap.OrganizationName, err)
	}
	return refreshed, nil
}

func currentOrgInfo(ctx context.Context, cfg *config.Config, token *auth.Token) *controlplane.OrgInfo {
	if cfg == nil || token == nil || strings.TrimSpace(token.AccessToken) == "" || token.OrganizationID() == "" {
		return &controlplane.OrgInfo{}
	}
	client := controlplane.NewClient(cfg.ControlPlaneBaseURL, token.AccessToken)
	info, err := client.GetOrg(ctx)
	if err != nil {
		return &controlplane.OrgInfo{}
	}
	return info
}

func shouldPromptForDisplayName(orgName, orgID string) bool {
	name := strings.TrimSpace(orgName)
	if name == "" {
		return true
	}
	if strings.EqualFold(name, strings.TrimSpace(orgID)) {
		return true
	}
	return strings.HasSuffix(strings.ToLower(name), " personal")
}
