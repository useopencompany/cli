package cmd

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/auth"
	"go.agentprotocol.cloud/cli/internal/config"
	"go.agentprotocol.cloud/cli/internal/controlplane"
	"go.agentprotocol.cloud/cli/internal/tui/spawn"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List operator sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		token, err := auth.EnsureValid(cfg)
		if err != nil {
			return fmt.Errorf("not authenticated — run 'ap auth login' first")
		}

		client := controlplane.NewClient(cfg.ControlPlaneBaseURL, token.AccessToken)
		sessions, err := client.ListSessions(cmd.Context())
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions yet. Run 'ap spawn' to start one.")
			return nil
		}

		fmt.Printf("%-44s  %-12s  %-19s  %s\n", "ID", "STATUS", "UPDATED", "TITLE")
		for _, s := range sessions {
			updated := s.UpdatedAt
			if updated.IsZero() {
				updated = s.CreatedAt
			}
			fmt.Printf("%-44s  %-12s  %-19s  %s\n", s.ID, s.Status, updated.Local().Format(time.DateTime), s.Title)
		}
		return nil
	},
}

var sessionCmd = &cobra.Command{
	Use:   "session <ID>",
	Short: "Open an existing operator session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		token, err := auth.EnsureValid(cfg)
		if err != nil {
			return fmt.Errorf("not authenticated — run 'ap auth login' first")
		}

		client := controlplane.NewClient(cfg.ControlPlaneBaseURL, token.AccessToken)
		session, messages, err := client.GetSession(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if session.RuntimeStatus != "ready" || session.Status != "ready" {
			return fmt.Errorf("ap runtime recovery is not supported yet for session %s", session.ID)
		}

		m := spawn.NewResumeModel(token.WorkspaceName(), client, session.ID, messages)
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sessionsCmd)
	rootCmd.AddCommand(sessionCmd)
}
