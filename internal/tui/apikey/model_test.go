package apikey

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

const testAnthropicKey = "sk-ant-12345678901234567890"

func TestChooseStorageDefaultsToKeyVault(t *testing.T) {
	model := NewStorageModel(testAnthropicKey, "https://agentplatform.cloud/docs/architecture", "")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	done, ok := msg.(DoneMsg)
	if !ok {
		t.Fatalf("expected DoneMsg, got %T", msg)
	}

	next := updated.(Model)
	if next.step != stepDone {
		t.Fatalf("expected stepDone, got %v", next.step)
	}
	if !done.InVault {
		t.Fatalf("expected key vault to be the default selection")
	}
}

func TestChooseStorageAlwaysSavesToVault(t *testing.T) {
	model := NewStorageModel(testAnthropicKey, "https://agentplatform.cloud/docs/architecture", "")

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	updated, cmd = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	done, ok := msg.(DoneMsg)
	if !ok {
		t.Fatalf("expected DoneMsg, got %T", msg)
	}

	next := updated.(Model)
	if next.step != stepDone {
		t.Fatalf("expected stepDone, got %v", next.step)
	}
	if !done.InVault {
		t.Fatalf("expected the key to be saved in the vault")
	}
}

func TestAutomaticSearchFallbackShowsManualNotice(t *testing.T) {
	model := NewModel("")
	model.step = stepSearching

	updated, cmd := model.Update(searchResultMsg{})
	if cmd != nil {
		t.Fatalf("expected no command, got %v", cmd)
	}

	next := updated.(Model)
	if next.step != stepManualEntry {
		t.Fatalf("expected stepManualEntry, got %v", next.step)
	}
	if next.manualNotice == "" {
		t.Fatalf("expected manual fallback notice to be set")
	}
	if view := next.View(); !strings.Contains(view, "ap couldn't find a key automatically.") {
		t.Fatalf("expected fallback notice in view, got %q", view)
	}
}
