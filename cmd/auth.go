package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/auth"
	"go.agentprotocol.cloud/cli/internal/config"
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

		if err := auth.SaveToken(token); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}

		fmt.Println("\n✓ Authenticated successfully.")

		// Show user info from token claims.
		if userID := token.UserID(); userID != "" {
			fmt.Printf("  User:      %s\n", userID)
		}
		if ws := token.WorkspaceName(); ws != "" {
			fmt.Printf("  Workspace: %s\n", ws)
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

		fmt.Println("✓ Authenticated")
		if userID := token.UserID(); userID != "" {
			fmt.Printf("  User:      %s\n", userID)
		}
		if ws := token.WorkspaceName(); ws != "" {
			fmt.Printf("  Workspace: %s\n", ws)
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
