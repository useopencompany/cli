package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	workosBaseURL      = "https://api.workos.com"
	deviceAuthEndpoint = "/user_management/authorize/device"
	tokenEndpoint      = "/user_management/authenticate"

	deviceGrantType  = "urn:ietf:params:oauth:grant-type:device_code"
	refreshGrantType = "refresh_token"
)

// DeviceAuthResponse is returned by the device authorization endpoint.
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// TokenResponse is returned by the token endpoint on success.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// tokenErrorResponse is returned by the token endpoint during polling.
type tokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// DeviceFlow runs the OAuth 2.0 Device Authorization Flow (RFC 8628) against
// WorkOS AuthKit. It opens the browser for the user and polls until the user
// completes authentication or the code expires.
func DeviceFlow(ctx context.Context, clientID string) (*Token, error) {
	// Step 1: Request device code.
	authResp, err := requestDeviceCode(ctx, clientID)
	if err != nil {
		return nil, fmt.Errorf("requesting device code: %w", err)
	}

	// Step 2: Prompt user and open browser.
	fmt.Printf("\nOpen this URL in your browser:\n\n")
	fmt.Printf("  %s\n\n", authResp.VerificationURI)
	fmt.Printf("Then enter this code: %s\n\n", authResp.UserCode)

	if authResp.VerificationURIComplete != "" {
		_ = openBrowser(authResp.VerificationURIComplete)
	}

	fmt.Println("Waiting for authentication...")

	// Step 3: Poll for token.
	interval := time.Duration(authResp.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}

	deadline := time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("device code expired — run 'ap auth login' again")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		tokenResp, pollErr := pollToken(ctx, clientID, authResp.DeviceCode)
		if pollErr != nil {
			switch pollErr.Error() {
			case "authorization_pending":
				// Expected — user hasn't completed auth yet.
				continue
			case "slow_down":
				// Back off by adding 5 seconds per RFC 8628 §3.5.
				interval += 5 * time.Second
				continue
			case "access_denied":
				return nil, fmt.Errorf("authorization denied by user")
			case "expired_token":
				return nil, fmt.Errorf("device code expired — run 'ap auth login' again")
			default:
				return nil, fmt.Errorf("polling for token: %w", pollErr)
			}
		}

		return &Token{
			AccessToken:  tokenResp.AccessToken,
			RefreshToken: tokenResp.RefreshToken,
			ExpiresAt:    expiresAtFromJWT(tokenResp.AccessToken, tokenResp.ExpiresIn),
		}, nil
	}
}

// RefreshAccessToken exchanges a refresh token for a new access token.
func RefreshAccessToken(clientID, refreshToken, organizationID string) (*Token, error) {
	form := url.Values{
		"grant_type":    {refreshGrantType},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	if strings.TrimSpace(organizationID) != "" {
		form.Set("organization_id", strings.TrimSpace(organizationID))
	}

	resp, err := http.Post(workosBaseURL+tokenEndpoint, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("refresh failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding refresh response: %w", err)
	}

	newToken := &Token{
		AccessToken: result.AccessToken,
		ExpiresAt:   expiresAtFromJWT(result.AccessToken, result.ExpiresIn),
	}
	// Use new refresh token if rotated, otherwise keep the old one.
	if result.RefreshToken != "" {
		newToken.RefreshToken = result.RefreshToken
	} else {
		newToken.RefreshToken = refreshToken
	}

	return newToken, nil
}

func requestDeviceCode(ctx context.Context, clientID string) (*DeviceAuthResponse, error) {
	form := url.Values{"client_id": {clientID}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, workosBaseURL+deviceAuthEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device auth failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result DeviceAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func pollToken(ctx context.Context, clientID, deviceCode string) (*TokenResponse, error) {
	form := url.Values{
		"grant_type":  {deviceGrantType},
		"device_code": {deviceCode},
		"client_id":   {clientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, workosBaseURL+tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Parse the error response to get the specific error code.
		var errResp tokenErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, errors.New(errResp.Error)
		}
		return nil, fmt.Errorf("token request failed (status %d)", resp.StatusCode)
	}

	var result TokenResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func expiresAtFromJWT(accessToken string, expiresIn int) time.Time {
	parts := strings.Split(accessToken, ".")
	if len(parts) == 3 {
		if payload, err := base64.RawURLEncoding.DecodeString(parts[1]); err == nil {
			var claims struct {
				Exp int64 `json:"exp"`
			}
			if json.Unmarshal(payload, &claims) == nil && claims.Exp > 0 {
				return time.Unix(claims.Exp, 0)
			}
		}
	}
	if expiresIn > 0 {
		return time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	return time.Now().Add(5 * time.Minute)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
