package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Discover and install interactive agents",
}

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List interactive agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		agents, err := client.ListAgents(cmd.Context())
		if err != nil {
			return err
		}
		if len(agents) == 0 {
			fmt.Println("No agents found.")
			return nil
		}
		fmt.Printf("%-34s  %-9s  %-6s  %s\n", "ID", "INSTALLED", "READY", "NAME")
		for _, agent := range agents {
			fmt.Printf("%-34s  %-9t  %-6t  %s\n", agent.ID, agent.Readiness.Installed, agentReady(agent), agent.Name)
		}
		return nil
	},
}

var agentsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search interactive agents",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		agents, err := client.FindAgents(cmd.Context(), controlplane.FindAgentsRequest{Query: strings.TrimSpace(args[0])})
		if err != nil {
			return err
		}
		if len(agents) == 0 {
			fmt.Println("No matching agents found.")
			return nil
		}
		fmt.Printf("%-34s  %-9s  %-6s  %s\n", "ID", "INSTALLED", "READY", "NAME")
		for _, agent := range agents {
			fmt.Printf("%-34s  %-9t  %-6t  %s\n", agent.ID, agent.Readiness.Installed, agentReady(agent), agent.Name)
		}
		return nil
	},
}

var agentsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show an interactive agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		agent, err := client.GetAgent(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		printAgentDetails(agent)
		return nil
	},
}

var agentsInstallCmd = &cobra.Command{
	Use:   "install <id>",
	Short: "Install an interactive agent into the active workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		agent, err := client.InstallAgent(cmd.Context(), controlplane.InstallAgentRequest{ID: strings.TrimSpace(args[0])})
		if err != nil {
			return err
		}
		fmt.Printf("Installed %s (%s)\n\n", agent.ID, agent.InstalledVersion)
		printAgentDetails(agent)
		fmt.Printf("\nNext: %s\n", agentInstallNextStep(agent))
		return nil
	},
}

func init() {
	agentsCmd.AddCommand(agentsListCmd)
	agentsCmd.AddCommand(agentsSearchCmd)
	agentsCmd.AddCommand(agentsShowCmd)
	agentsCmd.AddCommand(agentsInstallCmd)
	rootCmd.AddCommand(agentsCmd)
}

func printAgentDetails(agent *controlplane.Agent) {
	if agent == nil {
		return
	}
	fmt.Printf("ID:          %s\n", agent.ID)
	fmt.Printf("Name:        %s\n", agent.Name)
	fmt.Printf("Version:     %s\n", agent.Version)
	fmt.Printf("Installed:   %t\n", agent.Readiness.Installed)
	if strings.TrimSpace(agent.InstalledVersion) != "" {
		fmt.Printf("Pinned:      %s\n", agent.InstalledVersion)
		if strings.TrimSpace(agent.InstalledVersion) != strings.TrimSpace(agent.Version) {
			fmt.Printf("Refresh:     reinstall to update from %s to %s\n", agent.InstalledVersion, agent.Version)
		}
	}
	if strings.TrimSpace(agent.Category) != "" {
		fmt.Printf("Category:    %s\n", agent.Category)
	}
	if strings.TrimSpace(agent.Description) != "" {
		fmt.Printf("Description: %s\n", agent.Description)
	}
	if len(agent.Skills) > 0 {
		fmt.Printf("Skills:      %s\n", strings.Join(agent.Skills, ", "))
	}
	if len(agent.RequiredIntegrations) > 0 {
		fmt.Printf("Integrations:%s\n", " "+strings.Join(agent.RequiredIntegrations, ", "))
	}
	if len(agent.RecommendedFor) > 0 {
		fmt.Printf("Use cases:   %s\n", strings.Join(agent.RecommendedFor, ", "))
	}
	if agent.BootstrapCompletedAt != nil {
		fmt.Printf("Bootstrap:   completed at %s\n", agent.BootstrapCompletedAt.Local().Format("2006-01-02 15:04:05"))
	} else if agent.Readiness.Installed {
		fmt.Println("Bootstrap:   pending")
	}
	fmt.Printf("Ready:       %t\n", agentReady(*agent))
	if len(agent.Readiness.MissingSkills) > 0 {
		fmt.Printf("Missing skills:      %s\n", strings.Join(agent.Readiness.MissingSkills, ", "))
	}
	if len(agent.Readiness.MissingConnections) > 0 {
		fmt.Printf("Missing connections: %s\n", strings.Join(agent.Readiness.MissingConnections, ", "))
	}
	if len(agent.Readiness.MissingPermissions) > 0 {
		fmt.Printf("Missing permissions: %s\n", strings.Join(agent.Readiness.MissingPermissions, ", "))
	}
}

func agentReady(agent controlplane.Agent) bool {
	return len(agent.Readiness.MissingSkills) == 0 &&
		len(agent.Readiness.MissingConnections) == 0 &&
		len(agent.Readiness.MissingPermissions) == 0 &&
		(strings.TrimSpace(agent.InstalledVersion) == "" || strings.TrimSpace(agent.InstalledVersion) == strings.TrimSpace(agent.Version))
}

func agentInstallNextStep(agent *controlplane.Agent) string {
	if agent == nil {
		return "ap agents list"
	}
	if agentReady(*agent) {
		return "ap spawn --agent " + agent.ID
	}
	return "connect the missing integrations/permissions above, then run ap agents show " + agent.ID
}
