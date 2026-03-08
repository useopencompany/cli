package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

var permissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage explicit permission grants",
}

var (
	permissionsListWorkspaceID string
	permissionsListSubjectType string
	permissionsListSubjectID   string
)

var permissionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List permission grants",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		grants, err := client.ListPermissionGrants(cmd.Context(), permissionsListWorkspaceID, permissionsListSubjectType, permissionsListSubjectID)
		if err != nil {
			return err
		}
		if len(grants) == 0 {
			fmt.Println("No permission grants found.")
			return nil
		}
		fmt.Printf("%-44s  %-14s  %-24s  %-24s  %-24s\n", "ID", "SUBJECT", "SUBJECT_ID", "ACTION", "RESOURCE")
		for _, grant := range grants {
			fmt.Printf("%-44s  %-14s  %-24s  %-24s  %-24s\n", grant.ID, grant.SubjectType, grant.SubjectID, grant.Action, grant.Resource)
		}
		return nil
	},
}

var (
	permissionsGrantWorkspaceID string
	permissionsGrantSubjectType string
	permissionsGrantSubjectID   string
	permissionsGrantAction      string
	permissionsGrantResource    string
)

var permissionsGrantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Create an explicit permission grant",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		if strings.TrimSpace(permissionsGrantSubjectType) == "" || strings.TrimSpace(permissionsGrantSubjectID) == "" || strings.TrimSpace(permissionsGrantAction) == "" || strings.TrimSpace(permissionsGrantResource) == "" {
			return fmt.Errorf("--subject-type, --subject-id, --action, and --resource are required")
		}
		grant, err := client.GrantPermission(cmd.Context(), controlplane.GrantPermissionRequest{
			WorkspaceID: strings.TrimSpace(permissionsGrantWorkspaceID),
			SubjectType: strings.TrimSpace(permissionsGrantSubjectType),
			SubjectID:   strings.TrimSpace(permissionsGrantSubjectID),
			Action:      strings.TrimSpace(permissionsGrantAction),
			Resource:    strings.TrimSpace(permissionsGrantResource),
		})
		if err != nil {
			return err
		}
		fmt.Printf("Granted %s on %s to %s/%s: %s\n", grant.Action, grant.Resource, grant.SubjectType, grant.SubjectID, grant.ID)
		return nil
	},
}

var permissionsRevokeCmd = &cobra.Command{
	Use:   "revoke <grant-id>",
	Short: "Delete a permission grant",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		if err := client.RevokePermission(cmd.Context(), strings.TrimSpace(args[0])); err != nil {
			return err
		}
		fmt.Printf("Revoked permission grant %s\n", strings.TrimSpace(args[0]))
		return nil
	},
}

func init() {
	permissionsListCmd.Flags().StringVar(&permissionsListWorkspaceID, "workspace-id", "", "Filter by workspace")
	permissionsListCmd.Flags().StringVar(&permissionsListSubjectType, "subject-type", "", "Filter by subject type (user|org_role)")
	permissionsListCmd.Flags().StringVar(&permissionsListSubjectID, "subject-id", "", "Filter by subject id")

	permissionsGrantCmd.Flags().StringVar(&permissionsGrantWorkspaceID, "workspace-id", "", "Workspace scope for the grant")
	permissionsGrantCmd.Flags().StringVar(&permissionsGrantSubjectType, "subject-type", "", "Subject type (user|org_role)")
	permissionsGrantCmd.Flags().StringVar(&permissionsGrantSubjectID, "subject-id", "", "Subject identifier")
	permissionsGrantCmd.Flags().StringVar(&permissionsGrantAction, "action", "", "Permission action")
	permissionsGrantCmd.Flags().StringVar(&permissionsGrantResource, "resource", "", "Permission resource")

	permissionsCmd.AddCommand(permissionsListCmd)
	permissionsCmd.AddCommand(permissionsGrantCmd)
	permissionsCmd.AddCommand(permissionsRevokeCmd)
	rootCmd.AddCommand(permissionsCmd)
}
