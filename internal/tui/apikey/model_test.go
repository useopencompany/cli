package apikey

import (
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

func TestChooseStorageCanSwitchToLocal(t *testing.T) {
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
	if done.InVault {
		t.Fatalf("expected local storage after moving selection down")
	}
}
