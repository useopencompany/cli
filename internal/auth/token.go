package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.agentprotocol.cloud/cli/internal/config"
)

const credentialsFile = "credentials.json"

// Token represents stored authentication credentials.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// jwtClaims holds the subset of JWT claims we care about.
type jwtClaims struct {
	Sub   string `json:"sub"`    // WorkOS user ID (user_01...)
	OrgID string `json:"org_id"` // WorkOS organization ID
	Email string `json:"email"`  // User email (if present in token)
	SID   string `json:"sid"`    // Session ID
	Exp   int64  `json:"exp"`    // Expiration timestamp
	Iss   string `json:"iss"`    // Issuer
}

// IsExpired returns true if the access token has expired.
func (t *Token) IsExpired() bool {
	// Treat as expired 30s before actual expiry to avoid edge cases.
	return time.Now().After(t.ExpiresAt.Add(-30 * time.Second))
}

// Claims parses the JWT access token and returns the embedded claims.
// This does NOT verify the signature — we trust the token because we
// received it directly from WorkOS over TLS.
func (t *Token) Claims() (*jwtClaims, error) {
	parts := strings.Split(t.AccessToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed JWT: expected 3 parts, got %d", len(parts))
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding JWT payload: %w", err)
	}

	var claims jwtClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("parsing JWT claims: %w", err)
	}

	return &claims, nil
}

// WorkspaceName extracts the organization/workspace name from the token.
// Falls back to the org ID or a default if claims are unavailable.
func (t *Token) WorkspaceName() string {
	claims, err := t.Claims()
	if err != nil {
		return "Unknown Workspace"
	}

	if claims.OrgID != "" {
		return claims.OrgID
	}

	return "Personal"
}

// OrganizationID returns the WorkOS organization ID from the token claims.
func (t *Token) OrganizationID() string {
	claims, err := t.Claims()
	if err != nil {
		return ""
	}
	return claims.OrgID
}

// UserID returns the WorkOS user ID from the token claims.
func (t *Token) UserID() string {
	claims, err := t.Claims()
	if err != nil {
		return ""
	}
	return claims.Sub
}

// EnsureValid checks if the token is still valid. If expired but a refresh
// token is available, it attempts to refresh. Returns the (possibly refreshed)
// token and whether it's usable.
func EnsureValid(cfg *config.Config) (*Token, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("no stored credentials — run 'ap auth login'")
	}

	if !token.IsExpired() {
		return token, nil
	}

	// Token expired — try refresh.
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("session expired and no refresh token — run 'ap auth login'")
	}

	refreshed, err := RefreshAccessToken(cfg.WorkOSClientID, token.RefreshToken, "")
	if err != nil {
		return nil, fmt.Errorf("token refresh failed — run 'ap auth login': %w", err)
	}

	if err := SaveToken(refreshed); err != nil {
		return nil, fmt.Errorf("saving refreshed token: %w", err)
	}

	return refreshed, nil
}

// SaveToken persists the token in the OS-specific ap config directory.
func SaveToken(t *Token) error {
	dir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, credentialsFile), data, 0o600)
}

// LoadToken reads the stored token from disk.
func LoadToken() (*Token, error) {
	dir, err := config.Dir()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(dir, credentialsFile))
	if err != nil {
		return nil, err
	}

	var t Token
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}

	return &t, nil
}

// ClearToken removes stored credentials.
func ClearToken() error {
	dir, err := config.Dir()
	if err != nil {
		return err
	}

	path := filepath.Join(dir, credentialsFile)
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
