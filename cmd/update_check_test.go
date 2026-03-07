package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMaybeWarnIfOutdatedWarnsWhenBehindLatestRelease(t *testing.T) {
	originalVersion := version
	originalArgs := osArgs
	originalEnabled := updateCheckEnabled
	originalNow := updateCheckNow
	originalClient := updateCheckClient
	originalURL := updateCheckURL
	originalUserCacheDir := userCacheDir
	t.Cleanup(func() {
		version = originalVersion
		osArgs = originalArgs
		updateCheckEnabled = originalEnabled
		updateCheckNow = originalNow
		updateCheckClient = originalClient
		updateCheckURL = originalURL
		userCacheDir = originalUserCacheDir
	})

	version = "v0.2.0"
	osArgs = []string{"/tmp/ap"}
	updateCheckEnabled = func() bool { return true }
	updateCheckNow = func() time.Time {
		return time.Date(2026, time.March, 7, 10, 0, 0, 0, time.UTC)
	}
	cacheRoot := t.TempDir()
	userCacheDir = func() (string, error) { return cacheRoot, nil }

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/releases/latest"; got != want {
			t.Fatalf("request path = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.3.0"}`))
	}))
	defer server.Close()

	updateCheckClient = server.Client()
	updateCheckClient.Timeout = updateCheckTimeout
	updateCheckURL = server.URL + "/releases/latest"

	var stderr bytes.Buffer
	maybeWarnIfOutdated(&stderr)

	output := stderr.String()
	if !strings.Contains(output, "warning: ap v0.2.0 is outdated; latest release is v0.3.0") {
		t.Fatalf("warning output = %q", output)
	}

	cachePath := filepath.Join(cacheRoot, "ap", updateCheckCacheFile)
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cache file at %s: %v", cachePath, err)
	}
}

func TestMaybeWarnIfOutdatedSkipsNonComparableVersions(t *testing.T) {
	originalVersion := version
	originalEnabled := updateCheckEnabled
	t.Cleanup(func() {
		version = originalVersion
		updateCheckEnabled = originalEnabled
	})

	version = "dev+c52142752b03-dirty"
	updateCheckEnabled = func() bool { return true }

	var stderr bytes.Buffer
	maybeWarnIfOutdated(&stderr)

	if got := stderr.String(); got != "" {
		t.Fatalf("warning output = %q, want empty", got)
	}
}

func TestLatestReleaseVersionFallsBackToCachedValueOnFetchError(t *testing.T) {
	originalNow := updateCheckNow
	originalClient := updateCheckClient
	originalURL := updateCheckURL
	originalUserCacheDir := userCacheDir
	t.Cleanup(func() {
		updateCheckNow = originalNow
		updateCheckClient = originalClient
		updateCheckURL = originalURL
		userCacheDir = originalUserCacheDir
	})

	cacheRoot := t.TempDir()
	userCacheDir = func() (string, error) { return cacheRoot, nil }
	updateCheckNow = func() time.Time {
		return time.Date(2026, time.March, 7, 11, 0, 0, 0, time.UTC)
	}

	cachePath := filepath.Join(cacheRoot, "ap", updateCheckCacheFile)
	if err := saveReleaseCheckCache(cachePath, updateCheckCache{
		CheckedAt:     updateCheckNow().Add(-48 * time.Hour),
		LatestVersion: "v0.3.0",
	}); err != nil {
		t.Fatalf("saveReleaseCheckCache() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	updateCheckClient = server.Client()
	updateCheckClient.Timeout = updateCheckTimeout
	updateCheckURL = server.URL + "/releases/latest"

	latest, err := latestReleaseVersion(context.Background())
	if err != nil {
		t.Fatalf("latestReleaseVersion() error = %v", err)
	}
	if got, want := latest, "v0.3.0"; got != want {
		t.Fatalf("latestReleaseVersion() = %q, want %q", got, want)
	}
}
