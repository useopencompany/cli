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
	DefaultWorkOSClientID = "client_01KH38V2H563FEHATV5F0AS5SX"

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

// Dir returns the config directory path (~/.config/ap/).
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

// Load reads the config from disk. Missing values are populated from
// environment variables, then from compiled-in defaults.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	path := filepath.Join(dir, configFile)
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, cfg) // best-effort; fall through to defaults
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
