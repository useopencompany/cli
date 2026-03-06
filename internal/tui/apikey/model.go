package apikey

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DoneMsg is sent when the API key flow completes.
type DoneMsg struct {
	Key     string
	InVault bool // true = stored in ap vault, false = local only.
}

// step tracks where we are in the API key setup flow.
type step int

const (
	stepChooseSource step = iota
	stepSearching
	stepManualEntry
	stepFoundKey
	stepChooseStorage
	stepDone
)

// Model handles the interactive API key setup.
type Model struct {
	step         step
	cursor       int
	width        int
	key          string
	input        string
	foundKey     string
	inVault      bool
	docsURL      string
	manualNotice string
	validErr     string // validation error for manual entry
	storeErr     string
}

// NewModel creates a new API key setup model.
func NewModel(docsURL string) Model {
	return Model{step: stepChooseSource, docsURL: strings.TrimSpace(docsURL)}
}

// NewStorageModel returns the storage choice step with an existing key.
func NewStorageModel(key, docsURL, storeErr string) Model {
	return Model{
		step:     stepChooseStorage,
		key:      strings.TrimSpace(key),
		docsURL:  strings.TrimSpace(docsURL),
		storeErr: strings.TrimSpace(storeErr),
	}
}

// SetWidth updates the model width for layout.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

type searchResultMsg struct {
	key string
	err error
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case searchResultMsg:
		if msg.err != nil || msg.key == "" {
			m.step = stepManualEntry
			m.manualNotice = "ap couldn't find a key automatically."
			return m, nil
		}
		m.foundKey = msg.key
		m.manualNotice = ""
		m.step = stepFoundKey
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepChooseSource:
		return m.handleChooseSource(msg)
	case stepManualEntry:
		return m.handleManualEntry(msg)
	case stepFoundKey:
		return m.handleFoundKey(msg)
	case stepChooseStorage:
		return m.handleChooseStorage(msg)
	}
	return m, nil
}

func (m Model) handleChooseSource(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 1 {
			m.cursor++
		}
	case "enter":
		if m.cursor == 0 {
			m.step = stepSearching
			return m, m.searchForKey()
		}
		m.manualNotice = ""
		m.step = stepManualEntry
	}
	return m, nil
}

func (m Model) handleManualEntry(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		key := strings.TrimSpace(m.input)
		if key == "" {
			return m, nil
		}
		if err := validateAPIKey(key); err != "" {
			m.validErr = err
			return m, nil
		}
		m.key = key
		m.validErr = ""
		m.step = stepChooseStorage
		m.cursor = 0
		m.storeErr = ""
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
			m.validErr = "" // clear error on edit
		}
	default:
		// Accept both single keystrokes and pasted text (e.g. pasting an API key).
		// bubbletea delivers paste as a single KeyRunes msg with all characters.
		if msg.Type == tea.KeyRunes {
			m.input += string(msg.Runes)
			m.validErr = "" // clear error on edit
		}
	}
	return m, nil
}

func (m Model) handleFoundKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		m.key = m.foundKey
		m.step = stepChooseStorage
		m.cursor = 0
		m.storeErr = ""
	case "n", "N":
		m.manualNotice = ""
		m.step = stepManualEntry
	}
	return m, nil
}

func (m Model) handleChooseStorage(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < 1 {
			m.cursor++
		}
	case "enter":
		m.inVault = m.cursor == 0
		m.step = stepDone
		return m, func() tea.Msg {
			return DoneMsg{Key: m.key, InVault: m.inVault}
		}
	}
	return m, nil
}

// validateAPIKey checks that the key looks like a valid Anthropic API key.
func validateAPIKey(key string) string {
	if !strings.HasPrefix(key, "sk-ant-") {
		return "Key must start with sk-ant-"
	}
	if len(key) < 20 {
		return "Key is too short"
	}
	return ""
}

// searchForKey looks for ANTHROPIC_API_KEY in the environment and shell history.
func (m Model) searchForKey() tea.Cmd {
	return func() tea.Msg {
		// 1. Check active environment variable.
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return searchResultMsg{key: key}
		}

		// 2. Search shell history files.
		home, err := os.UserHomeDir()
		if err != nil {
			return searchResultMsg{err: err}
		}

		histFiles := []string{
			home + "/.bash_history",
			home + "/.zsh_history",
		}

		for _, hf := range histFiles {
			key := searchHistoryFile(hf)
			if key != "" {
				return searchResultMsg{key: key}
			}
		}

		// 3. Try `printenv` as fallback.
		out, err := exec.Command("printenv", "ANTHROPIC_API_KEY").Output()
		if err == nil {
			key := strings.TrimSpace(string(out))
			if key != "" {
				return searchResultMsg{key: key}
			}
		}

		return searchResultMsg{}
	}
}

func searchHistoryFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	// Search backwards for most recent occurrence.
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		// Match patterns like: export ANTHROPIC_API_KEY=sk-ant-...
		if idx := strings.Index(line, "ANTHROPIC_API_KEY="); idx != -1 {
			val := line[idx+len("ANTHROPIC_API_KEY="):]
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			if strings.HasPrefix(val, "sk-ant-") {
				return val
			}
		}
	}

	return ""
}

// View implements tea.Model.
func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Orange.
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))    // Red.
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)

	var b strings.Builder
	b.WriteString("\n")

	switch m.step {
	case stepChooseSource:
		b.WriteString(titleStyle.Render("  No ANTHROPIC_API_KEY found."))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("  How would you like to provide one?"))
		b.WriteString("\n\n")

		options := []string{
			"Search automatically (env vars, shell history)",
			"Enter it manually",
		}
		for i, opt := range options {
			if i == m.cursor {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("  ▸ %s", opt)))
			} else {
				b.WriteString(dimStyle.Render(fmt.Sprintf("    %s", opt)))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  ↑/↓ navigate • enter select"))

	case stepSearching:
		b.WriteString(dimStyle.Render("  Searching for API key..."))

	case stepManualEntry:
		b.WriteString(titleStyle.Render("  Enter your Anthropic API key:"))
		b.WriteString("\n\n")
		if m.manualNotice != "" {
			b.WriteString(warnStyle.Render(fmt.Sprintf("  %s", m.manualNotice)))
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("  > %s█", m.input))
		b.WriteString("\n")
		if m.validErr != "" {
			b.WriteString(errStyle.Render(fmt.Sprintf("    %s", m.validErr)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  enter: confirm • paste your sk-ant-... key"))

	case stepFoundKey:
		b.WriteString(titleStyle.Render("  Found an API key:"))
		b.WriteString("\n\n")
		masked := maskKey(m.foundKey)
		b.WriteString(fmt.Sprintf("  %s", masked))
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("  Use this key? (y/n)"))

	case stepChooseStorage:
		b.WriteString(titleStyle.Render("  Where should this key be stored?"))
		b.WriteString("\n\n")

		if m.storeErr != "" {
			b.WriteString(errStyle.Render(fmt.Sprintf("  %s", m.storeErr)))
			b.WriteString("\n\n")
		}

		options := []string{
			"ap key vault (recommended)",
			"Keep locally",
		}
		for i, opt := range options {
			if i == m.cursor {
				b.WriteString(selectedStyle.Render(fmt.Sprintf("  ▸ %s", opt)))
			} else {
				b.WriteString(dimStyle.Render(fmt.Sprintf("    %s", opt)))
			}
			b.WriteString("\n")
		}

		b.WriteString("\n")
		if m.cursor == 0 {
			b.WriteString(dimStyle.Render("  Saved inside Agent Platform so ap can reuse it across sessions."))
			if m.docsURL != "" {
				b.WriteString("\n")
				b.WriteString(dimStyle.Render(fmt.Sprintf("  Docs: %s", m.docsURL)))
			}
		} else {
			b.WriteString(warnStyle.Render("  ⚠ ap will not remember this key across sessions."))
			b.WriteString("\n")
			b.WriteString(warnStyle.Render("    The ANTHROPIC_API_KEY env var must be set each time."))
		}
		b.WriteString("\n\n")
		b.WriteString(dimStyle.Render("  ↑/↓ navigate • enter confirm"))

	case stepDone:
		if m.inVault {
			b.WriteString(dimStyle.Render("  Saving key to ap key vault..."))
		} else {
			b.WriteString(dimStyle.Render("  ✓ Key set for this session."))
		}
	}

	return b.String()
}

// maskKey shows the first 10 and last 4 characters of an API key.
func maskKey(key string) string {
	if len(key) <= 14 {
		return key[:4] + "..." + key[len(key)-2:]
	}
	return key[:10] + "..." + key[len(key)-4:]
}
