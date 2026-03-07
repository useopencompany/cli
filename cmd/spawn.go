package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/auth"
	"go.agentprotocol.cloud/cli/internal/config"
	"go.agentprotocol.cloud/cli/internal/controlplane"
	"go.agentprotocol.cloud/cli/internal/tui/spawn"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn",
	Short: "Launch an operator session",
	Long:  "Opens an interactive TUI for interacting with Operator.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Ensure authentication — tries refresh if token is expired.
		token, err := auth.EnsureValid(cfg)
		if err != nil {
			return fmt.Errorf("not authenticated — run 'ap auth login' first")
		}

		client := controlplane.NewClient(cfg.ControlPlaneBaseURL, token.AccessToken)
		workspace := token.WorkspaceName()
		if info, infoErr := fetchNamedOrgInfo(cmd.Context(), client); infoErr == nil {
			workspace = namedContextLabel(info, workspace)
		}
		docsURL := apKeyVaultDocsURL(cfg)

		m := spawn.NewModel(workspace, client, docsURL, spawnAgentID)
		p := tea.NewProgram(m, tea.WithAltScreen())

		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		return nil
	},
}

var spawnAgentID string

func init() {
	spawnCmd.Flags().StringVar(&spawnAgentID, "agent", "", "Start the session with an installed interactive agent")
	rootCmd.AddCommand(spawnCmd)
}
