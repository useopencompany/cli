package cmd

import "testing"

func TestDashboardURLBuildsDashboardPath(t *testing.T) {
	target, err := dashboardURL("https://agentplatform.cloud")
	if err != nil {
		t.Fatalf("dashboardURL returned error: %v", err)
	}

	if got, want := target.String(), "https://agentplatform.cloud/dashboard"; got != want {
		t.Fatalf("dashboardURL() = %q, want %q", got, want)
	}
}

func TestDashboardURLDropsExistingQueryAndFragment(t *testing.T) {
	target, err := dashboardURL("https://agentplatform.cloud/app?from=cli#section")
	if err != nil {
		t.Fatalf("dashboardURL returned error: %v", err)
	}

	if got, want := target.String(), "https://agentplatform.cloud/app/dashboard"; got != want {
		t.Fatalf("dashboardURL() = %q, want %q", got, want)
	}
}

func TestDashboardURLErrorsOnEmptyBaseURL(t *testing.T) {
	if _, err := dashboardURL("   "); err == nil {
		t.Fatal("dashboardURL() error = nil, want error")
	}
}
