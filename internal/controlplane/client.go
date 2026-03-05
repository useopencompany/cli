package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL     string
	accessToken string
	http        *http.Client
}

func NewClient(baseURL, accessToken string) *Client {
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		http:        &http.Client{Timeout: 10 * time.Minute},
	}
}

type APIError struct {
	Status int
	Method string
	Path   string
	Code   string
	Msg    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Code) != "" {
		return fmt.Sprintf("control-plane %s %s failed with %d (%s): %s", e.Method, e.Path, e.Status, e.Code, e.Msg)
	}
	return fmt.Sprintf("control-plane %s %s failed with %d: %s", e.Method, e.Path, e.Status, e.Msg)
}

type Session struct {
	ID               string    `json:"id"`
	Status           string    `json:"status"`
	RuntimeStatus    string    `json:"runtime_status"`
	RecoveryStatus   string    `json:"recovery_status"`
	RecoveryError    string    `json:"recovery_error"`
	RecoveryAttempts int       `json:"recovery_attempts"`
	Title            string    `json:"title"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateSessionRequest struct {
	AnthropicAPIKey string `json:"anthropic_api_key"`
	Title           string `json:"title,omitempty"`
}

type CreateTurnRequest struct {
	Content string `json:"content"`
}

type TurnResponse struct {
	TurnID           string `json:"turn_id"`
	Status           string `json:"status"`
	AssistantContent string `json:"assistant_content"`
	Error            string `json:"error"`
	Code             string `json:"code"`
}

type listSessionsResponse struct {
	Sessions []Session `json:"sessions"`
}

type getSessionResponse struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
}

type Workspace struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrgInfo struct {
	OrgID           string    `json:"org_id"`
	UserSub         string    `json:"user_sub"`
	Role            string    `json:"role"`
	ActiveWorkspace Workspace `json:"active_workspace"`
}

type InviteOrgMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

type InviteOrgMemberResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	OrganizationID string    `json:"organization_id"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type ListWorkspacesResponse struct {
	Workspaces      []Workspace `json:"workspaces"`
	ActiveWorkspace Workspace   `json:"active_workspace"`
}

type CreateWorkspaceRequest struct {
	Name string `json:"name"`
}

type SwitchWorkspaceRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type IntegrationConnection struct {
	ID            string    `json:"id"`
	OrgID         string    `json:"org_id"`
	WorkspaceID   string    `json:"workspace_id"`
	OwnerUserSub  string    `json:"owner_user_sub"`
	Scope         string    `json:"scope"`
	Integration   string    `json:"integration"`
	Provider      string    `json:"provider"`
	CredentialRef string    `json:"credential_ref"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type CreateIntegrationConnectionRequest struct {
	Integration   string            `json:"integration"`
	Provider      string            `json:"provider"`
	Scope         string            `json:"scope"`
	WorkspaceID   string            `json:"workspace_id,omitempty"`
	CredentialRef string            `json:"credential_ref,omitempty"`
	Credentials   map[string]string `json:"credentials"`
}

type UpdateIntegrationConnectionRequest struct {
	Integration   *string           `json:"integration,omitempty"`
	Provider      *string           `json:"provider,omitempty"`
	Scope         *string           `json:"scope,omitempty"`
	WorkspaceID   *string           `json:"workspace_id,omitempty"`
	CredentialRef *string           `json:"credential_ref,omitempty"`
	Status        *string           `json:"status,omitempty"`
	Credentials   map[string]string `json:"credentials,omitempty"`
}

type ListIntegrationConnectionsResponse struct {
	Connections []IntegrationConnection `json:"connections"`
}

type Action struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	Integration        string `json:"integration"`
	Provider           string `json:"provider"`
	ReadOnly           bool   `json:"read_only"`
	PermissionAction   string `json:"permission_action"`
	PermissionResource string `json:"permission_resource"`
}

type ListActionsResponse struct {
	Actions []Action `json:"actions"`
}

type FindActionsRequest struct {
	Query       string `json:"query,omitempty"`
	Integration string `json:"integration,omitempty"`
	Provider    string `json:"provider,omitempty"`
}

type ExecuteActionRequest struct {
	Action       string         `json:"action"`
	Input        map[string]any `json:"input,omitempty"`
	ConnectionID string         `json:"connection_id,omitempty"`
	SessionID    string         `json:"session_id,omitempty"`
}

type ExecuteActionResponse struct {
	InvocationID string         `json:"invocation_id"`
	Action       string         `json:"action"`
	ConnectionID string         `json:"connection_id"`
	Result       map[string]any `json:"result"`
}

type ActionInvocation struct {
	ID             string          `json:"id"`
	SessionID      string          `json:"session_id"`
	OrgID          string          `json:"org_id"`
	WorkspaceID    string          `json:"workspace_id"`
	UserSub        string          `json:"user_sub"`
	Action         string          `json:"action"`
	Resource       string          `json:"resource"`
	ConnectionID   string          `json:"connection_id"`
	PermissionUsed string          `json:"permission_used"`
	Status         string          `json:"status"`
	DurationMS     int64           `json:"duration_ms"`
	Cost           string          `json:"cost"`
	Error          string          `json:"error"`
	CreatedAt      time.Time       `json:"created_at"`
	Request        json.RawMessage `json:"sanitized_request"`
	Response       json.RawMessage `json:"sanitized_response"`
}

type ListActionInvocationsResponse struct {
	Invocations []ActionInvocation `json:"invocations"`
}

func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	var out Session
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/sessions", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListSessions(ctx context.Context) ([]Session, error) {
	var out listSessionsResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/sessions", nil, &out); err != nil {
		return nil, err
	}
	return out.Sessions, nil
}

func (c *Client) GetSession(ctx context.Context, id string) (*Session, []Message, error) {
	var out getSessionResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/sessions/"+id, nil, &out); err != nil {
		return nil, nil, err
	}
	return &out.Session, out.Messages, nil
}

func (c *Client) CreateTurn(ctx context.Context, sessionID, content string) (*TurnResponse, error) {
	var out TurnResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/sessions/"+sessionID+"/turns", CreateTurnRequest{Content: content}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTurn(ctx context.Context, sessionID, turnID string) (*TurnResponse, error) {
	var out TurnResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/sessions/"+sessionID+"/turns/"+turnID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SetRecoveryKey(ctx context.Context, sessionID, key string) error {
	return c.doJSON(ctx, http.MethodPost, "/v1/operator/sessions/"+sessionID+"/recovery-key", map[string]string{
		"anthropic_api_key": key,
	}, nil)
}

func (c *Client) WaitForSessionReady(ctx context.Context, sessionID string, timeout time.Duration) (*Session, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		session, _, err := c.GetSession(ctx, sessionID)
		if err == nil {
			if session.Status == "ready" && session.RuntimeStatus == "ready" {
				return session, nil
			}
			if session.Status == "failed" {
				return nil, fmt.Errorf("session provisioning failed")
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("session provisioning timed out")
		case <-ticker.C:
		}
	}
}

func (c *Client) GetOrg(ctx context.Context) (*OrgInfo, error) {
	var out OrgInfo
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/org", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) InviteOrgMember(ctx context.Context, req InviteOrgMemberRequest) (*InviteOrgMemberResponse, error) {
	var out InviteOrgMemberResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/org/invite", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListWorkspaces(ctx context.Context) (*ListWorkspacesResponse, error) {
	var out ListWorkspacesResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/workspaces", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (*Workspace, error) {
	var out Workspace
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/workspaces", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SwitchWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	var out Workspace
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/workspaces/switch", SwitchWorkspaceRequest{WorkspaceID: workspaceID}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateIntegrationConnection(ctx context.Context, req CreateIntegrationConnectionRequest) (*IntegrationConnection, error) {
	var out IntegrationConnection
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/integrations/connections", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListIntegrationConnections(ctx context.Context, integration, provider string, includeRevoked bool) ([]IntegrationConnection, error) {
	path := "/v1/operator/integrations/connections"
	query := make([]string, 0, 3)
	if strings.TrimSpace(integration) != "" {
		query = append(query, "integration="+urlQueryEscape(integration))
	}
	if strings.TrimSpace(provider) != "" {
		query = append(query, "provider="+urlQueryEscape(provider))
	}
	if includeRevoked {
		query = append(query, "include_revoked=true")
	}
	if len(query) > 0 {
		path += "?" + strings.Join(query, "&")
	}
	var out ListIntegrationConnectionsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Connections, nil
}

func (c *Client) UpdateIntegrationConnection(ctx context.Context, id string, req UpdateIntegrationConnectionRequest) (*IntegrationConnection, error) {
	var out IntegrationConnection
	if err := c.doJSON(ctx, http.MethodPatch, "/v1/operator/integrations/connections/"+id, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteIntegrationConnection(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/v1/operator/integrations/connections/"+id, nil, nil)
}

func (c *Client) ListActions(ctx context.Context, integration, provider string) ([]Action, error) {
	path := "/v1/operator/actions"
	query := make([]string, 0, 2)
	if strings.TrimSpace(integration) != "" {
		query = append(query, "integration="+urlQueryEscape(integration))
	}
	if strings.TrimSpace(provider) != "" {
		query = append(query, "provider="+urlQueryEscape(provider))
	}
	if len(query) > 0 {
		path += "?" + strings.Join(query, "&")
	}
	var out ListActionsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Actions, nil
}

func (c *Client) FindActions(ctx context.Context, req FindActionsRequest) ([]Action, error) {
	var out ListActionsResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/actions/find", req, &out); err != nil {
		return nil, err
	}
	return out.Actions, nil
}

func (c *Client) ExecuteAction(ctx context.Context, req ExecuteActionRequest) (*ExecuteActionResponse, error) {
	var out ExecuteActionResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/actions/execute", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListActionInvocations(ctx context.Context, all bool, limit int) ([]ActionInvocation, error) {
	path := "/v1/operator/actions/invocations"
	query := make([]string, 0, 2)
	if all {
		query = append(query, "all=true")
	}
	if limit > 0 {
		query = append(query, "limit="+fmt.Sprintf("%d", limit))
	}
	if len(query) > 0 {
		path += "?" + strings.Join(query, "&")
	}
	var out ListActionInvocationsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Invocations, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, respBody any) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		if len(raw) == 0 {
			return fmt.Errorf("control-plane %s %s failed with %d", method, path, resp.StatusCode)
		}
		var payload struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}
		if err := json.Unmarshal(raw, &payload); err == nil && strings.TrimSpace(payload.Error) != "" {
			return &APIError{
				Status: resp.StatusCode,
				Method: method,
				Path:   path,
				Code:   payload.Code,
				Msg:    payload.Error,
			}
		}
		return &APIError{
			Status: resp.StatusCode,
			Method: method,
			Path:   path,
			Msg:    strings.TrimSpace(string(raw)),
		}
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return err
		}
	}
	return nil
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"+", "%2B",
		"&", "%26",
		"=", "%3D",
		"?", "%3F",
		"#", "%23",
		"/", "%2F",
	)
	return replacer.Replace(strings.TrimSpace(value))
}
