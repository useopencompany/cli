package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type testItem struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestJSONObject(t *testing.T) {
	var buf bytes.Buffer
	item := testItem{Name: "alpha", Value: 42}
	if err := JSONTo(&buf, item); err != nil {
		t.Fatalf("JSONTo returned error: %v", err)
	}
	out := buf.String()
	var got testItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if got != item {
		t.Errorf("got %+v, want %+v", got, item)
	}
}

func TestJSONArray(t *testing.T) {
	var buf bytes.Buffer
	items := []testItem{{Name: "a", Value: 1}, {Name: "b", Value: 2}}
	if err := JSONTo(&buf, items); err != nil {
		t.Fatalf("JSONTo returned error: %v", err)
	}
	out := buf.String()
	var got []testItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &got); err != nil {
		t.Fatalf("output is not valid JSON array: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d items, want 2", len(got))
	}
}

func TestJSONEmptySlice(t *testing.T) {
	var buf bytes.Buffer
	items := make([]string, 0)
	if err := JSONTo(&buf, items); err != nil {
		t.Fatalf("JSONTo returned error: %v", err)
	}
	out := strings.TrimSuffix(buf.String(), "\n")
	if out != "[]" {
		t.Errorf("got %q, want %q", out, "[]")
	}
}

func TestJSONNilSlice(t *testing.T) {
	var buf bytes.Buffer
	var items []string // nil slice
	if err := JSONTo(&buf, items); err != nil {
		t.Fatalf("JSONTo returned error: %v", err)
	}
	out := strings.TrimSuffix(buf.String(), "\n")
	if out != "[]" {
		t.Errorf("got %q, want %q (nil slice must produce [] not null)", out, "[]")
	}
}

func TestJSONCompact(t *testing.T) {
	var buf bytes.Buffer
	item := testItem{Name: "test", Value: 99}
	if err := JSONTo(&buf, item); err != nil {
		t.Fatalf("JSONTo returned error: %v", err)
	}
	out := strings.TrimSuffix(buf.String(), "\n")
	if strings.Contains(out, "\n") || strings.Contains(out, "  ") {
		t.Errorf("output should be compact, got %q", out)
	}
}

func TestJSONTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	item := testItem{Name: "test", Value: 1}
	if err := JSONTo(&buf, item); err != nil {
		t.Fatalf("JSONTo returned error: %v", err)
	}
	out := buf.String()
	if !strings.HasSuffix(out, "\n") {
		t.Errorf("output should end with newline, got %q", out)
	}
	// Should end with exactly one newline
	if strings.HasSuffix(out, "\n\n") {
		t.Errorf("output should end with exactly one newline, got %q", out)
	}
}

func TestJSONToWriter(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]string{"key": "value"}
	if err := JSONTo(&buf, data); err != nil {
		t.Fatalf("JSONTo returned error: %v", err)
	}
	out := strings.TrimSuffix(buf.String(), "\n")
	if out != `{"key":"value"}` {
		t.Errorf("got %q, want %q", out, `{"key":"value"}`)
	}
}

func TestJSONMarshalError(t *testing.T) {
	var buf bytes.Buffer
	ch := make(chan int) // channels cannot be marshaled
	err := JSONTo(&buf, ch)
	if err == nil {
		t.Error("expected error for unmarshalable type, got nil")
	}
}
