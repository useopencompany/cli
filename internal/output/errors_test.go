package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"go.agentprotocol.cloud/cli/internal/controlplane"
)

func TestErrorJSONPlainError(t *testing.T) {
	var buf bytes.Buffer
	ErrorJSON(&buf, fmt.Errorf("something broke"))
	out := strings.TrimSuffix(buf.String(), "\n")
	want := `{"error":"something broke"}`
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestErrorJSONAPIError(t *testing.T) {
	var buf bytes.Buffer
	apiErr := &controlplane.APIError{
		Status: 404,
		Code:   "not_found",
		Msg:    "agent not found",
	}
	ErrorJSON(&buf, apiErr)
	out := strings.TrimSuffix(buf.String(), "\n")
	want := `{"error":"agent not found","code":"not_found","status":404}`
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestErrorJSONAPIErrorEmptyCode(t *testing.T) {
	var buf bytes.Buffer
	apiErr := &controlplane.APIError{
		Status: 500,
		Code:   "",
		Msg:    "internal error",
	}
	ErrorJSON(&buf, apiErr)
	out := strings.TrimSuffix(buf.String(), "\n")
	// Code should be omitted when empty
	if strings.Contains(out, `"code"`) {
		t.Errorf("output should omit empty code field, got %q", out)
	}
	// Status should still be present
	if !strings.Contains(out, `"status":500`) {
		t.Errorf("output should contain status, got %q", out)
	}
}

func TestErrorJSONWrappedAPIError(t *testing.T) {
	var buf bytes.Buffer
	apiErr := &controlplane.APIError{
		Status: 403,
		Code:   "forbidden",
		Msg:    "access denied",
	}
	wrapped := fmt.Errorf("request failed: %w", apiErr)
	ErrorJSON(&buf, wrapped)
	out := strings.TrimSuffix(buf.String(), "\n")
	want := `{"error":"access denied","code":"forbidden","status":403}`
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestErrorJSONCompactNewline(t *testing.T) {
	var buf bytes.Buffer
	ErrorJSON(&buf, fmt.Errorf("test"))
	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("output should end with newline, got %q", out)
	}
	trimmed := strings.TrimSuffix(out, "\n")
	if strings.Contains(trimmed, "\n") || strings.Contains(trimmed, "  ") {
		t.Errorf("output should be compact single-line, got %q", out)
	}
}
