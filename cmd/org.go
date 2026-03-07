package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

var orgCmd = &cobra.Command{
	Use:   "org",
	Short: "Show organization info",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		org, err := client.GetOrg(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Printf("Org:       %s\n", org.OrgID)
		fmt.Printf("User:      %s\n", org.UserSub)
		fmt.Printf("Role:      %s\n", org.Role)
		fmt.Printf("Workspace: %s (%s)\n", org.ActiveWorkspace.Name, org.ActiveWorkspace.ID)
		return nil
	},
}

var (
	orgInviteEmail string
	orgInviteRole  string
)

var orgInviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Invite a user to your organization",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		if orgInviteEmail == "" {
			return fmt.Errorf("--email is required")
		}
		req := controlplane.InviteOrgMemberRequest{
			Email: orgInviteEmail,
			Role:  orgInviteRole,
		}
		resp, err := client.InviteOrgMember(cmd.Context(), req)
		if err != nil {
			return err
		}
		fmt.Printf("Invite created: %s\n", resp.ID)
		fmt.Printf("Email:          %s\n", resp.Email)
		fmt.Printf("Status:         %s\n", resp.Status)
		if !resp.ExpiresAt.IsZero() {
			fmt.Printf("Expires:        %s\n", resp.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

func init() {
	orgInviteCmd.Flags().StringVar(&orgInviteEmail, "email", "", "Invitee email")
	orgInviteCmd.Flags().StringVar(&orgInviteRole, "role", "member", "Role for invitee (member|admin)")
	orgCmd.AddCommand(orgInviteCmd)
	rootCmd.AddCommand(orgCmd)
}
