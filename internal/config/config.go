package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	appName    = "ap"
	configFile = "config.json"

	// DefaultWorkOSClientID is the production WorkOS client ID shared across
	// dev and production environments (see opencompany AGENTS.md).
	DefaultWorkOSClientID = "client_01KH38V370P81BFBSTPXGRZ82B"

	// DefaultWorkOSAuthDomain is the custom AuthKit domain. Both staging and
	// production JWTs use this as the issuer (verified empirically).
	DefaultWorkOSAuthDomain = "auth.opencompany.cloud"

	// DefaultControlPlaneBaseURL is the production ap control-plane API.
	DefaultControlPlaneBaseURL = "https://ap-controlplane.fly.dev"

	// DefaultDashboardBaseURL is the production dashboard URL.
	DefaultDashboardBaseURL = "https://agentplatform.cloud"
)

// Config holds application-level configuration.
type Config struct {
	WorkOSClientID      string `json:"workos_client_id"`
	WorkOSAuthDomain    string `json:"workos_auth_domain,omitempty"`
	ControlPlaneBaseURL string `json:"control_plane_base_url,omitempty"`
	DashboardBaseURL    string `json:"dashboard_base_url,omitempty"`
}

// Dir returns the OS-specific config directory path for ap.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, appName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	return dir, nil
}

// Load returns the runtime configuration for the CLI.
//
// Production connectivity/auth settings always default to the compiled-in
// production values. They may only be overridden via environment variables.
// Persisted config.json values are intentionally ignored here so a stale local
// file cannot pin production installs to a dev control plane.
func Load() (*Config, error) {
	if _, err := Dir(); err != nil {
		return nil, err
	}

	cfg := &Config{
		WorkOSClientID:      DefaultWorkOSClientID,
		WorkOSAuthDomain:    DefaultWorkOSAuthDomain,
		ControlPlaneBaseURL: DefaultControlPlaneBaseURL,
		DashboardBaseURL:    DefaultDashboardBaseURL,
	}

	applyEnvOverride(&cfg.WorkOSClientID, "AP_WORKOS_CLIENT_ID", DefaultWorkOSClientID)
	applyEnvOverride(&cfg.WorkOSAuthDomain, "AP_WORKOS_AUTH_DOMAIN", DefaultWorkOSAuthDomain)
	applyEnvOverride(&cfg.ControlPlaneBaseURL, "AP_CONTROL_PLANE_BASE_URL", DefaultControlPlaneBaseURL)
	applyEnvOverride(&cfg.DashboardBaseURL, "AP_DASHBOARD_BASE_URL", DefaultDashboardBaseURL)

	return cfg, nil
}

// Save writes the config to disk.
func Save(cfg *Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, configFile), data, 0o600)
}

func applyEnvOverride(target *string, key, fallback string) {
	if v := os.Getenv(key); v != "" {
		*target = v
		return
	}
	if *target == "" {
		*target = fallback
	}
}
