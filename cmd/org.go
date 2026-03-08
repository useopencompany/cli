package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/auth"
	"go.agentprotocol.cloud/cli/internal/config"
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

		fmt.Printf("Organization: %s\n", firstNonEmpty(org.OrgName, org.OrgID))
		if displayName := strings.TrimSpace(org.UserDisplayName); displayName != "" {
			fmt.Printf("User:         %s\n", displayName)
		} else {
			fmt.Printf("User:         %s\n", org.UserSub)
		}
		fmt.Printf("Role:         %s\n", org.Role)
		fmt.Printf("Workspace:    %s\n", org.ActiveWorkspace.Name)
		return nil
	},
}

var (
	orgInviteEmail  string
	orgInviteRole   string
	orgCreateName   string
	orgCreateSwitch bool
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
		fmt.Printf("Invite created for %s\n", resp.Email)
		fmt.Printf("Status: %s\n", resp.Status)
		if !resp.ExpiresAt.IsZero() {
			fmt.Printf("Expires: %s\n", resp.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List organizations you can access",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		orgs, err := client.ListOrganizationMemberships(cmd.Context())
		if err != nil {
			return err
		}
		if len(orgs) == 0 {
			fmt.Println("No organizations found.")
			return nil
		}
		fmt.Printf("%-32s  %-8s  %-8s  %s\n", "NAME", "ROLE", "STATUS", "CURRENT")
		for _, org := range orgs {
			current := ""
			if org.Current {
				current = "yes"
			}
			fmt.Printf("%-32s  %-8s  %-8s  %s\n", org.Name, org.Role, org.Status, current)
		}
		return nil
	},
}

var orgCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new organization",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, token, client, err := authenticatedClient()
		if err != nil {
			return err
		}

		name := strings.TrimSpace(orgCreateName)
		if name == "" {
			name, err = promptForName("Organization name")
			if err != nil {
				return err
			}
		}

		org, err := client.CreateOrganization(cmd.Context(), controlplane.CreateOrganizationRequest{Name: name})
		if err != nil {
			return err
		}
		fmt.Printf("Created organization %s\n", org.Name)

		if orgCreateSwitch {
			return switchOrganization(cmd, cfg, token, org.OrgID, org.Name)
		}
		return nil
	},
}

var orgSwitchCmd = &cobra.Command{
	Use:   "switch <organization>",
	Short: "Switch the active organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, token, client, err := authenticatedClient()
		if err != nil {
			return err
		}

		orgs, err := client.ListOrganizationMemberships(cmd.Context())
		if err != nil {
			return err
		}
		org, err := resolveOrganization(args[0], orgs)
		if err != nil {
			return err
		}

		return switchOrganization(cmd, cfg, token, org.OrgID, org.Name)
	},
}

func init() {
	orgInviteCmd.Flags().StringVar(&orgInviteEmail, "email", "", "Invitee email")
	orgInviteCmd.Flags().StringVar(&orgInviteRole, "role", "member", "Role for invitee (member|admin)")
	orgCreateCmd.Flags().StringVar(&orgCreateName, "name", "", "Organization name")
	orgCreateCmd.Flags().BoolVar(&orgCreateSwitch, "switch", true, "Switch into the new organization after creating it")

	orgCmd.AddCommand(orgCreateCmd)
	orgCmd.AddCommand(orgInviteCmd)
	orgCmd.AddCommand(orgListCmd)
	orgCmd.AddCommand(orgSwitchCmd)
	rootCmd.AddCommand(orgCmd)
}

func resolveOrganization(input string, orgs []controlplane.OrganizationMembership) (*controlplane.OrganizationMembership, error) {
	target := strings.TrimSpace(input)
	if target == "" {
		return nil, fmt.Errorf("organization is required")
	}

	for _, org := range orgs {
		if strings.EqualFold(strings.TrimSpace(org.OrgID), target) {
			match := org
			return &match, nil
		}
	}

	var matches []controlplane.OrganizationMembership
	for _, org := range orgs {
		if strings.EqualFold(strings.TrimSpace(org.Name), target) {
			matches = append(matches, org)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("organization %q not found", target)
	case 1:
		return &matches[0], nil
	default:
		return nil, fmt.Errorf("organization name %q is ambiguous", target)
	}
}

func switchOrganization(cmd *cobra.Command, cfg *config.Config, token *auth.Token, orgID, orgName string) error {
	refreshed, err := auth.RefreshAccessToken(cfg.WorkOSClientID, token.RefreshToken, orgID)
	if err != nil {
		return err
	}
	if err := auth.SaveToken(refreshed); err != nil {
		return err
	}

	client := controlplane.NewClient(cfg.ControlPlaneBaseURL, refreshed.AccessToken)
	info, err := client.GetOrg(cmd.Context())
	if err != nil {
		return err
	}

	fmt.Printf("Active organization: %s\n", firstNonEmpty(info.OrgName, orgName))
	fmt.Printf("Active workspace: %s\n", info.ActiveWorkspace.Name)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
