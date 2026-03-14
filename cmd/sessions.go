package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/auth"
	"go.agentprotocol.cloud/cli/internal/config"
	"go.agentprotocol.cloud/cli/internal/controlplane"
	"go.agentprotocol.cloud/cli/internal/tui/spawn"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List sessions",
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
	Short: "Open an existing session",
	Args:  validateSessionArgs,
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
			return sessionLookupError(cmd, args[0], err)
		}
		workspace := token.WorkspaceName()
		if info, infoErr := fetchNamedOrgInfo(cmd.Context(), client); infoErr == nil {
			workspace = namedContextLabel(info, workspace)
		}

		m := spawn.NewResumeModel(workspace, client, session.ID, messages)
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

func validateSessionArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(cmd, args); err != nil {
		return err
	}
	if looksLikeSessionAction(args[0]) {
		return unknownSessionCommandError(cmd, args[0])
	}
	return nil
}

func sessionLookupError(cmd *cobra.Command, rawArg string, err error) error {
	if err == nil {
		return nil
	}
	if looksLikeSessionAction(rawArg) {
		return unknownSessionCommandError(cmd, rawArg)
	}

	var apiErr *controlplane.APIError
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
		if looksLikeSessionID(rawArg) {
			return fmt.Errorf("session %q not found\n\nRun '%s sessions' to list available sessions", rawArg, cmd.Root().Name())
		}
		return unknownSessionCommandError(cmd, rawArg)
	}
	return err
}

func unknownSessionCommandError(cmd *cobra.Command, rawArg string) error {
	binary := cmd.Root().Name()
	sessionPath := fmt.Sprintf("%s session", binary)
	return fmt.Errorf(
		"unknown command %q for %q\n\nDid you mean one of these?\n  %s spawn\n  %s sessions\n  %s session <ID>",
		rawArg,
		sessionPath,
		binary,
		binary,
		binary,
	)
}

func looksLikeSessionAction(rawArg string) bool {
	arg := strings.TrimSpace(rawArg)
	if arg == "" || looksLikeSessionID(arg) {
		return false
	}

	sawLetter := false
	for _, r := range arg {
		switch {
		case unicode.IsLetter(r):
			sawLetter = true
		case r == '-':
		default:
			return false
		}
	}
	return sawLetter
}

func looksLikeSessionID(rawArg string) bool {
	arg := strings.TrimSpace(rawArg)
	return strings.HasPrefix(arg, "sess_") && len(arg) > len("sess_")
}
