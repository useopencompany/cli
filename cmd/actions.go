package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

var (
	actionsIntegration string
	actionsProvider    string
)

var actionsCmd = &cobra.Command{
	Use:   "actions",
	Short: "List executable actions",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		actions, err := client.ListActions(cmd.Context(), actionsIntegration, actionsProvider)
		if err != nil {
			return err
		}
		if len(actions) == 0 {
			fmt.Println("No executable actions found.")
			return nil
		}
		fmt.Printf("%-34s  %-18s  %-28s\n", "ID", "INTEGRATION", "NAME")
		for _, action := range actions {
			fmt.Printf("%-34s  %-18s  %-28s\n", action.ID, action.Integration+"/"+action.Provider, action.Name)
		}
		return nil
	},
}

var (
	findIntegration string
	findProvider    string
)

var findCmd = &cobra.Command{
	Use:   "find <query>",
	Short: "Search executable actions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		actions, err := client.FindActions(cmd.Context(), controlplane.FindActionsRequest{
			Query:       strings.TrimSpace(args[0]),
			Integration: strings.TrimSpace(findIntegration),
			Provider:    strings.TrimSpace(findProvider),
		})
		if err != nil {
			return err
		}
		if len(actions) == 0 {
			fmt.Println("No matching actions found.")
			return nil
		}
		fmt.Printf("%-34s  %-18s  %-28s\n", "ID", "INTEGRATION", "NAME")
		for _, action := range actions {
			fmt.Printf("%-34s  %-18s  %-28s\n", action.ID, action.Integration+"/"+action.Provider, action.Name)
		}
		return nil
	},
}

var (
	doConnectionID string
	doSessionID    string
	doInputs       []string
	doJSONInput    string
)

var doCmd = &cobra.Command{
	Use:   "do <action-id>",
	Short: "Execute a read action through Platform Gateway",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}

		input := map[string]any{}
		if strings.TrimSpace(doJSONInput) != "" {
			if err := json.Unmarshal([]byte(doJSONInput), &input); err != nil {
				return fmt.Errorf("parsing --json payload: %w", err)
			}
		}
		pairs, err := parseInputPairs(doInputs)
		if err != nil {
			return err
		}
		for key, value := range pairs {
			input[key] = value
		}

		resp, err := client.ExecuteAction(cmd.Context(), controlplane.ExecuteActionRequest{
			Action:       strings.TrimSpace(args[0]),
			Input:        input,
			ConnectionID: strings.TrimSpace(doConnectionID),
			SessionID:    strings.TrimSpace(doSessionID),
		})
		if err != nil {
			return err
		}

		fmt.Printf("Invocation: %s\n", resp.InvocationID)
		fmt.Printf("Action:     %s\n", resp.Action)
		fmt.Printf("Connection: %s\n", resp.ConnectionID)
		pretty, _ := json.MarshalIndent(resp.Result, "", "  ")
		if len(pretty) > 0 {
			fmt.Println(string(pretty))
		}
		return nil
	},
}

var (
	invocationsAll   bool
	invocationsLimit int
)

var actionsInvocationsCmd = &cobra.Command{
	Use:   "invocations",
	Short: "List action invocation audit records",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, client, err := authenticatedClient()
		if err != nil {
			return err
		}
		rows, err := client.ListActionInvocations(cmd.Context(), invocationsAll, invocationsLimit)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			fmt.Println("No invocations found.")
			return nil
		}
		fmt.Printf("%-44s  %-10s  %-32s  %s\n", "ID", "STATUS", "ACTION", "CREATED")
		for _, row := range rows {
			fmt.Printf("%-44s  %-10s  %-32s  %s\n", row.ID, row.Status, row.Action, row.CreatedAt.Local().Format("2006-01-02 15:04:05"))
		}
		return nil
	},
}

func parseInputPairs(values []string) (map[string]any, error) {
	out := map[string]any{}
	for _, raw := range values {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		parts := strings.SplitN(item, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --input %q, expected key=value", item)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid --input %q, key is empty", item)
		}
		out[key] = value
	}

	return out, nil
}

func init() {
	actionsCmd.Flags().StringVar(&actionsIntegration, "integration", "", "Filter by integration")
	actionsCmd.Flags().StringVar(&actionsProvider, "provider", "", "Filter by provider")
	actionsCmd.AddCommand(actionsInvocationsCmd)

	actionsInvocationsCmd.Flags().BoolVar(&invocationsAll, "all", false, "Include all users in workspace (admin only)")
	actionsInvocationsCmd.Flags().IntVar(&invocationsLimit, "limit", 50, "Max rows to fetch")

	findCmd.Flags().StringVar(&findIntegration, "integration", "", "Filter by integration")
	findCmd.Flags().StringVar(&findProvider, "provider", "", "Filter by provider")

	doCmd.Flags().StringVar(&doConnectionID, "connection", "", "Explicit connection ID override")
	doCmd.Flags().StringVar(&doSessionID, "session", "", "Optional session ID for audit linking")
	doCmd.Flags().StringArrayVar(&doInputs, "input", nil, "Action input key=value (repeatable)")
	doCmd.Flags().StringVar(&doJSONInput, "json", "", "Action input JSON object")

	rootCmd.AddCommand(actionsCmd)
	rootCmd.AddCommand(findCmd)
	rootCmd.AddCommand(doCmd)
}
