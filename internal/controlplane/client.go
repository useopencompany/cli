package controlplane

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        string    `json:"id"`
	TurnID    string    `json:"turn_id,omitempty"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateSessionRequest struct {
	Title  string `json:"title,omitempty"`
	Source string `json:"source,omitempty"`
}

type CreateTurnRequest struct {
	Content string `json:"content"`
	Wait    *bool  `json:"wait,omitempty"`
}

type CreateRunRequest struct {
	Content string `json:"content,omitempty"`
}

type TurnResponse struct {
	TurnID           string    `json:"turn_id"`
	Status           string    `json:"status"`
	AssistantContent string    `json:"assistant_content"`
	Error            string    `json:"error"`
	Code             string    `json:"code"`
	Messages         []Message `json:"messages"`
}

type RunOutputBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type RunOutputMessage struct {
	Type    string           `json:"type"`
	Role    string           `json:"role"`
	Content []RunOutputBlock `json:"content"`
}

type ToolCall struct {
	ID            string         `json:"id"`
	ToolUseID     string         `json:"tool_use_id,omitempty"`
	Name          string         `json:"name"`
	Status        string         `json:"status"`
	Arguments     map[string]any `json:"arguments,omitempty"`
	ResultPreview map[string]any `json:"result_preview,omitempty"`
	Error         string         `json:"error,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type Run struct {
	ID        string             `json:"id"`
	Object    string             `json:"object"`
	SessionID string             `json:"session_id"`
	Status    string             `json:"status"`
	Source    string             `json:"source,omitempty"`
	Error     string             `json:"error,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
	Output    []RunOutputMessage `json:"output,omitempty"`
	ToolCalls []ToolCall         `json:"tool_calls,omitempty"`
}

type listRunsResponse struct {
	Runs []Run `json:"runs"`
}

type RunEvent struct {
	ID             string         `json:"id"`
	Type           string         `json:"type"`
	RunID          string         `json:"run_id"`
	SessionID      string         `json:"session_id"`
	SequenceNumber int            `json:"sequence_number"`
	CreatedAt      time.Time      `json:"created_at"`
	Data           map[string]any `json:"data"`
}

type listRunEventsResponse struct {
	Events []RunEvent `json:"events"`
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
	OrgName         string    `json:"org_name"`
	OrgID           string    `json:"org_id"`
	UserSub         string    `json:"user_sub"`
	UserDisplayName string    `json:"user_display_name"`
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

type OrganizationMembership struct {
	OrgID   string `json:"org_id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	Status  string `json:"status"`
	Current bool   `json:"current"`
}

type ListOrganizationMembershipsResponse struct {
	Organizations []OrganizationMembership `json:"organizations"`
}

type CreateOrganizationRequest struct {
	Name string `json:"name"`
}

type BootstrapRequest struct {
	PreferredOrganizationID string `json:"preferred_organization_id,omitempty"`
	DisplayName             string `json:"display_name,omitempty"`
}

type BootstrapResponse struct {
	OrganizationID              string    `json:"organization_id"`
	OrganizationName            string    `json:"organization_name"`
	UserDisplayName             string    `json:"user_display_name"`
	ActiveWorkspace             Workspace `json:"active_workspace"`
	CreatedPersonalOrganization bool      `json:"created_personal_organization"`
	UsingPersonalOrganization   bool      `json:"using_personal_organization"`
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
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	WorkspaceID  string    `json:"workspace_id"`
	OwnerUserSub string    `json:"owner_user_sub"`
	Scope        string    `json:"scope"`
	Integration  string    `json:"integration"`
	Provider     string    `json:"provider"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateIntegrationConnectionRequest struct {
	Integration string            `json:"integration"`
	Provider    string            `json:"provider"`
	Scope       string            `json:"scope"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	Credentials map[string]string `json:"credentials"`
}

type UpdateIntegrationConnectionRequest struct {
	Integration *string           `json:"integration,omitempty"`
	Provider    *string           `json:"provider,omitempty"`
	Scope       *string           `json:"scope,omitempty"`
	WorkspaceID *string           `json:"workspace_id,omitempty"`
	Status      *string           `json:"status,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
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
	ID             string    `json:"id"`
	SessionID      string    `json:"session_id"`
	OrgID          string    `json:"org_id"`
	WorkspaceID    string    `json:"workspace_id"`
	UserSub        string    `json:"user_sub"`
	Action         string    `json:"action"`
	Resource       string    `json:"resource"`
	ConnectionID   string    `json:"connection_id"`
	PermissionUsed string    `json:"permission_used"`
	Status         string    `json:"status"`
	DurationMS     int64     `json:"duration_ms"`
	Cost           string    `json:"cost"`
	InputKeys      []string  `json:"input_keys"`
	Error          string    `json:"error"`
	CreatedAt      time.Time `json:"created_at"`
}

type ListActionInvocationsResponse struct {
	Invocations []ActionInvocation `json:"invocations"`
}

type PermissionGrant struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	WorkspaceID string    `json:"workspace_id"`
	SubjectType string    `json:"subject_type"`
	SubjectID   string    `json:"subject_id"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListPermissionGrantsResponse struct {
	Grants []PermissionGrant `json:"grants"`
}

type GrantPermissionRequest struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	Action      string `json:"action"`
	Resource    string `json:"resource"`
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

func (c *Client) CreateTurn(ctx context.Context, sessionID string, req CreateTurnRequest) (*TurnResponse, error) {
	var out TurnResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/sessions/"+sessionID+"/turns", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateRun(ctx context.Context, sessionID string, req CreateRunRequest) (*Run, error) {
	var out Run
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/sessions/"+sessionID+"/runs", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetRun(ctx context.Context, runID string) (*Run, error) {
	var out Run
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/runs/"+runID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListRunEvents(ctx context.Context, runID string, startingAfter int) ([]RunEvent, error) {
	path := "/v1/operator/runs/" + runID + "/events"
	if startingAfter > 0 {
		path += "?starting_after=" + fmt.Sprintf("%d", startingAfter)
	}
	var out listRunEventsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Events, nil
}

func (c *Client) StreamRunEvents(ctx context.Context, runID string, startingAfter int) (<-chan RunEvent, <-chan error) {
	events := make(chan RunEvent)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		path := fmt.Sprintf("%s/v1/operator/runs/%s/events?stream=true&starting_after=%d", c.baseURL, runID, startingAfter)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		if err != nil {
			errs <- err
			return
		}
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Authorization", "Bearer "+c.accessToken)

		resp, err := c.http.Do(req)
		if err != nil {
			errs <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			raw, _ := io.ReadAll(resp.Body)
			errs <- &APIError{
				Status: resp.StatusCode,
				Method: http.MethodGet,
				Path:   "/v1/operator/runs/" + runID + "/events",
				Msg:    strings.TrimSpace(string(raw)),
			}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		const maxScanTokenSize = 1024 * 1024
		scanner.Buffer(make([]byte, 0, 64*1024), maxScanTokenSize)

		var dataLines []string
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if len(dataLines) > 0 {
					var event RunEvent
					if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &event); err == nil {
						select {
						case <-ctx.Done():
							return
						case events <- event:
						}
					}
					dataLines = dataLines[:0]
				}
				continue
			}
			if strings.HasPrefix(line, "data: ") {
				dataLines = append(dataLines, strings.TrimPrefix(line, "data: "))
			}
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
			errs <- err
		}
	}()

	return events, errs
}

func (c *Client) GetTurn(ctx context.Context, sessionID, turnID string) (*TurnResponse, error) {
	var out TurnResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/sessions/"+sessionID+"/turns/"+turnID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) WaitForSessionReady(ctx context.Context, sessionID string, timeout time.Duration) (*Session, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		session, _, err := c.GetSession(ctx, sessionID)
		if err == nil {
			if session.Status == "ready" {
				return session, nil
			}
			if session.Status == "failed" {
				return nil, fmt.Errorf("session startup failed")
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, fmt.Errorf("session startup timed out")
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

func (c *Client) Bootstrap(ctx context.Context, req BootstrapRequest) (*BootstrapResponse, error) {
	var out BootstrapResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/bootstrap", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListOrganizationMemberships(ctx context.Context) ([]OrganizationMembership, error) {
	var out ListOrganizationMembershipsResponse
	if err := c.doJSON(ctx, http.MethodGet, "/v1/operator/org/memberships", nil, &out); err != nil {
		return nil, err
	}
	return out.Organizations, nil
}

func (c *Client) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (*OrganizationMembership, error) {
	var out OrganizationMembership
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/org", req, &out); err != nil {
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

func (c *Client) ListPermissionGrants(ctx context.Context, workspaceID, subjectType, subjectID string) ([]PermissionGrant, error) {
	path := "/v1/operator/permissions/grants"
	query := make([]string, 0, 3)
	if strings.TrimSpace(workspaceID) != "" {
		query = append(query, "workspace_id="+urlQueryEscape(workspaceID))
	}
	if strings.TrimSpace(subjectType) != "" {
		query = append(query, "subject_type="+urlQueryEscape(subjectType))
	}
	if strings.TrimSpace(subjectID) != "" {
		query = append(query, "subject_id="+urlQueryEscape(subjectID))
	}
	if len(query) > 0 {
		path += "?" + strings.Join(query, "&")
	}
	var out ListPermissionGrantsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out.Grants, nil
}

func (c *Client) GrantPermission(ctx context.Context, req GrantPermissionRequest) (*PermissionGrant, error) {
	var out PermissionGrant
	if err := c.doJSON(ctx, http.MethodPost, "/v1/operator/permissions/grants", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RevokePermission(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/v1/operator/permissions/grants/"+id, nil, nil)
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
