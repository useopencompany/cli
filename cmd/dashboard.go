package cmd

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Open the web dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, token, _, err := authenticatedClient()
		if err != nil {
			return err
		}

		base := strings.TrimSpace(cfg.DashboardBaseURL)
		if base == "" {
			return fmt.Errorf("dashboard_base_url is empty")
		}

		target, err := url.Parse(strings.TrimRight(base, "/"))
		if err != nil {
			return fmt.Errorf("invalid dashboard URL: %w", err)
		}

		target.Path = strings.TrimRight(target.Path, "/") + "/auth/cli"
		query := target.Query()
		query.Set("access_token", token.AccessToken)
		query.Set("return_to", "/dashboard")
		target.RawQuery = query.Encode()

		if err := openBrowser(target.String()); err != nil {
			fmt.Println("Could not open browser automatically.")
			fmt.Printf("Open this URL:\n%s\n", target.String())
			return nil
		}

		fmt.Println("Dashboard opened in browser.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
