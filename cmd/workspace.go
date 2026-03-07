package cmd

import (
	"fmt"

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
		fmt.Printf("%-44s  %-24s  %-8s  %s\n", "ID", "NAME", "DEFAULT", "ACTIVE")
		for _, ws := range resp.Workspaces {
			active := ""
			if ws.ID == resp.ActiveWorkspace.ID {
				active = "yes"
			}
			fmt.Printf("%-44s  %-24s  %-8t  %s\n", ws.ID, ws.Name, ws.IsDefault, active)
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
		if workspaceCreateName == "" {
			return fmt.Errorf("--name is required")
		}
		ws, err := client.CreateWorkspace(cmd.Context(), controlplane.CreateWorkspaceRequest{Name: workspaceCreateName})
		if err != nil {
			return err
		}
		fmt.Printf("Created workspace %s (%s)\n", ws.Name, ws.ID)
		return nil
	},
}

var workspaceSwitchCmd = &cobra.Command{
	Use:   "switch <workspace-id>",
	Short: "Switch active workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		ws, err := client.SwitchWorkspace(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Active workspace: %s (%s)\n", ws.Name, ws.ID)
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
		fmt.Printf("%s (%s)\n", org.ActiveWorkspace.Name, org.ActiveWorkspace.ID)
		return nil
	},
}

func init() {
	workspaceCreateCmd.Flags().StringVar(&workspaceCreateName, "name", "", "Workspace name")

	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCmd.AddCommand(workspaceSwitchCmd)
	workspaceCmd.AddCommand(workspaceShowCmd)
	rootCmd.AddCommand(workspaceCmd)
}
