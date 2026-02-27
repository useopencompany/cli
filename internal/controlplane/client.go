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
		http:        &http.Client{Timeout: 120 * time.Second},
	}
}

type Session struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	RuntimeStatus string    `json:"runtime_status"`
	Title         string    `json:"title"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
}

type listSessionsResponse struct {
	Sessions []Session `json:"sessions"`
}

type getSessionResponse struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
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
		return fmt.Errorf("control-plane %s %s failed with %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	if respBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBody); err != nil {
			return err
		}
	}
	return nil
}
