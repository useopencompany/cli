package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvOverridesConfigFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("AP_WORKOS_CLIENT_ID", "env-client")
	t.Setenv("AP_WORKOS_AUTH_DOMAIN", "env-auth.example.com")
	t.Setenv("AP_CONTROL_PLANE_BASE_URL", "https://env-control-plane.example.com")
	t.Setenv("AP_DASHBOARD_BASE_URL", "https://env-dashboard.example.com")

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}

	payload := `{
  "workos_client_id": "file-client",
  "workos_auth_domain": "file-auth.example.com",
  "control_plane_base_url": "https://file-control-plane.example.com",
  "dashboard_base_url": "https://file-dashboard.example.com"
}`
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WorkOSClientID != "env-client" {
		t.Fatalf("WorkOSClientID = %q, want env override", cfg.WorkOSClientID)
	}
	if cfg.WorkOSAuthDomain != "env-auth.example.com" {
		t.Fatalf("WorkOSAuthDomain = %q, want env override", cfg.WorkOSAuthDomain)
	}
	if cfg.ControlPlaneBaseURL != "https://env-control-plane.example.com" {
		t.Fatalf("ControlPlaneBaseURL = %q, want env override", cfg.ControlPlaneBaseURL)
	}
	if cfg.DashboardBaseURL != "https://env-dashboard.example.com" {
		t.Fatalf("DashboardBaseURL = %q, want env override", cfg.DashboardBaseURL)
	}
}

func TestLoadIgnoresPersistedEndpointOverridesWithoutEnv(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error = %v", err)
	}

	payload := `{
  "workos_client_id": "file-client",
  "workos_auth_domain": "file-auth.example.com",
  "control_plane_base_url": "https://ap-controlplane-dev.fly.dev",
  "dashboard_base_url": "https://agentplatform-dev.example.com"
}`
	if err := os.WriteFile(filepath.Join(dir, configFile), []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.WorkOSClientID != DefaultWorkOSClientID {
		t.Fatalf("WorkOSClientID = %q, want built-in default", cfg.WorkOSClientID)
	}
	if cfg.WorkOSAuthDomain != DefaultWorkOSAuthDomain {
		t.Fatalf("WorkOSAuthDomain = %q, want built-in default", cfg.WorkOSAuthDomain)
	}
	if cfg.ControlPlaneBaseURL != DefaultControlPlaneBaseURL {
		t.Fatalf("ControlPlaneBaseURL = %q, want built-in default", cfg.ControlPlaneBaseURL)
	}
	if cfg.DashboardBaseURL != DefaultDashboardBaseURL {
		t.Fatalf("DashboardBaseURL = %q, want built-in default", cfg.DashboardBaseURL)
	}
}
