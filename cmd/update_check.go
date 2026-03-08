package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	updateCheckCacheFile = "release-check.json"
	updateCheckTTL       = 24 * time.Hour
	updateCheckTimeout   = 1500 * time.Millisecond
)

type updateCheckCache struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
}

var (
	updateCheckEnabled = defaultUpdateCheckEnabled
	updateCheckNow     = time.Now
	updateCheckClient  = &http.Client{Timeout: updateCheckTimeout}
	updateCheckURL     = "https://api.github.com/repos/useopencompany/cli/releases/latest"
	userCacheDir       = os.UserCacheDir
)

func maybeWarnIfOutdated(w io.Writer) {
	if w == nil || !updateCheckEnabled() {
		return
	}

	current := normalizedComparableVersion(resolvedVersion())
	if current == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	latest, err := latestReleaseVersion(ctx)
	if err != nil || latest == "" {
		return
	}
	if semver.Compare(current, latest) >= 0 {
		return
	}

	fmt.Fprintf(w, "warning: %s %s is outdated; latest release is %s. Run 'curl -fsSL https://agentplatform.cloud/install.sh | bash' to upgrade.\n", binaryName(), current, latest)
}

func defaultUpdateCheckEnabled() bool {
	if strings.TrimSpace(os.Getenv("AP_SKIP_UPDATE_CHECK")) != "" {
		return false
	}
	if len(osArgs) > 1 {
		switch strings.TrimSpace(osArgs[1]) {
		case "__complete", "__completeNoDesc", "completion":
			return false
		}
	}

	info, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func latestReleaseVersion(ctx context.Context) (string, error) {
	cachePath, cacheErr := releaseCheckCachePath()
	cached, _ := loadReleaseCheckCache(cachePath)
	if cached != nil {
		if latest := normalizedComparableVersion(cached.LatestVersion); latest != "" && updateCheckNow().Sub(cached.CheckedAt) < updateCheckTTL {
			return latest, nil
		}
	}

	latest, err := fetchLatestReleaseVersion(ctx)
	if err == nil {
		if cachePath != "" {
			_ = saveReleaseCheckCache(cachePath, updateCheckCache{
				CheckedAt:     updateCheckNow().UTC(),
				LatestVersion: latest,
			})
		}
		return latest, nil
	}

	if cached != nil {
		if latest := normalizedComparableVersion(cached.LatestVersion); latest != "" {
			return latest, nil
		}
	}
	if cacheErr != nil {
		return "", cacheErr
	}
	return "", err
}

func fetchLatestReleaseVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateCheckURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", binaryName(), resolvedVersion()))

	resp, err := updateCheckClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("latest release lookup returned %s", resp.Status)
	}

	var payload latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	latest := normalizedComparableVersion(payload.TagName)
	if latest == "" {
		return "", fmt.Errorf("latest release tag %q is not comparable", payload.TagName)
	}
	return latest, nil
}

func normalizedComparableVersion(raw string) string {
	version := strings.TrimSpace(raw)
	if version == "" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) {
		return ""
	}
	return version
}

func releaseCheckCachePath() (string, error) {
	base, err := userCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "ap", updateCheckCacheFile), nil
}

func loadReleaseCheckCache(path string) (*updateCheckCache, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("release check cache path is required")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cached updateCheckCache
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

func saveReleaseCheckCache(path string, cached updateCheckCache) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("release check cache path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
