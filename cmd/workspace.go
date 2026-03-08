package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage workspaces",
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		resp, err := client.ListWorkspaces(cmd.Context())
		if err != nil {
			return err
		}
		if len(resp.Workspaces) == 0 {
			fmt.Println("No workspaces found.")
			return nil
		}
		fmt.Printf("%-32s  %-8s  %s\n", "NAME", "DEFAULT", "ACTIVE")
		for _, ws := range resp.Workspaces {
			active := ""
			if ws.ID == resp.ActiveWorkspace.ID {
				active = "yes"
			}
			fmt.Printf("%-32s  %-8t  %s\n", ws.Name, ws.IsDefault, active)
		}
		return nil
	},
}

var workspaceCreateName string

var workspaceCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		name := strings.TrimSpace(workspaceCreateName)
		if name == "" {
			name, err = promptForName("Workspace name")
			if err != nil {
				return err
			}
		}
		ws, err := client.CreateWorkspace(cmd.Context(), controlplane.CreateWorkspaceRequest{Name: name})
		if err != nil {
			return err
		}
		fmt.Printf("Created workspace %s\n", ws.Name)
		return nil
	},
}

var workspaceSwitchCmd = &cobra.Command{
	Use:   "switch <workspace>",
	Short: "Switch active workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		resp, err := client.ListWorkspaces(cmd.Context())
		if err != nil {
			return err
		}
		target, err := resolveWorkspace(args[0], resp.Workspaces)
		if err != nil {
			return err
		}
		ws, err := client.SwitchWorkspace(cmd.Context(), target.ID)
		if err != nil {
			return err
		}
		fmt.Printf("Active workspace: %s\n", ws.Name)
		return nil
	},
}

var workspaceShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show active workspace",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		org, err := client.GetOrg(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Println(org.ActiveWorkspace.Name)
		return nil
	},
}

func resolveWorkspace(input string, workspaces []controlplane.Workspace) (*controlplane.Workspace, error) {
	target := strings.TrimSpace(input)
	if target == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	for _, workspace := range workspaces {
		if strings.EqualFold(strings.TrimSpace(workspace.ID), target) {
			match := workspace
			return &match, nil
		}
	}

	var matches []controlplane.Workspace
	for _, workspace := range workspaces {
		if strings.EqualFold(strings.TrimSpace(workspace.Name), target) {
			matches = append(matches, workspace)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("workspace %q not found", target)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("workspace name %q is ambiguous", target)
	}
}

func init() {
	workspaceCreateCmd.Flags().StringVar(&workspaceCreateName, "name", "", "Workspace name")

	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCmd.AddCommand(workspaceSwitchCmd)
	workspaceCmd.AddCommand(workspaceShowCmd)
	rootCmd.AddCommand(workspaceCmd)
}
