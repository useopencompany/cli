package spawn

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"go.agentprotocol.cloud/cli/internal/controlplane"
	"go.agentprotocol.cloud/cli/internal/tui/apikey"
)

// phase tracks where we are in the spawn lifecycle.
type phase int

const (
	phaseAPIKeyCheck phase = iota
	phaseAPIKeySetup
	phaseProvisionRuntime
	phaseOperator
)

// Message represents a single operator message.
type Message struct {
	Role    string
	Content string
}

// Model is the top-level TUI model for the spawn command.
type Model struct {
	workspace string
	phase     phase
	width     int
	height    int

	api                 *controlplane.Client
	sessionID           string
	apiKey              string
	pendingRetryContent string
	awaitingRecoveryKey bool

	// API key setup sub-model.
	apiKeyModel apikey.Model

	// Operator state.
	messages   []Message
	viewport   viewport.Model
	input      textarea.Model
	err        error
	submitting bool
}

// NewModel creates a spawn TUI for creating a new operator session.
func NewModel(workspace string, client *controlplane.Client) Model {
	return newModel(workspace, client, "", nil)
}

// NewResumeModel creates a spawn TUI for an existing operator session.
func NewResumeModel(workspace string, client *controlplane.Client, sessionID string, history []controlplane.Message) Model {
	messages := make([]Message, 0, len(history))
	for _, msg := range history {
		messages = append(messages, Message{Role: msg.Role, Content: msg.Content})
	}
	return newModel(workspace, client, sessionID, messages)
}

func newModel(workspace string, client *controlplane.Client, sessionID string, messages []Message) Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Prompt = "│ "
	ta.CharLimit = 4096
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = ta.FocusedStyle.CursorLine.Copy()
	ta.ShowLineNumbers = false
	ta.Focus()

	vp := viewport.New(80, 20)

	startPhase := phaseAPIKeyCheck
	apiKey := ""
	if sessionID != "" {
		startPhase = phaseOperator
	} else if envKey := os.Getenv("ANTHROPIC_API_KEY"); envKey != "" {
		apiKey = envKey
		startPhase = phaseProvisionRuntime
	}

	return Model{
		workspace:   workspace,
		phase:       startPhase,
		api:         client,
		sessionID:   sessionID,
		apiKey:      apiKey,
		apiKeyModel: apikey.NewModel(),
		messages:    messages,
		viewport:    vp,
		input:       ta,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	switch m.phase {
	case phaseAPIKeyCheck:
		return func() tea.Msg { return apiKeyMissingMsg{} }
	case phaseProvisionRuntime:
		return m.createSessionCmd()
	case phaseOperator:
		return textarea.Blink
	default:
		return nil
	}
}

type apiKeyMissingMsg struct{}
type sessionCreatedMsg struct{ sessionID string }
type sessionReadyMsg struct{}
type sessionErrorMsg struct{ err error }
type turnDoneMsg struct{ content string }
type turnErrorMsg struct{ err error }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 3
		inputHeight := 5
		operatorHeight := m.height - headerHeight - inputHeight
		if operatorHeight < 1 {
			operatorHeight = 1
		}
		m.viewport.Width = m.width
		m.viewport.Height = operatorHeight
		m.input.SetWidth(m.width - 2)
		m.apiKeyModel.SetWidth(m.width)
		return m, nil

	case apiKeyMissingMsg:
		m.phase = phaseAPIKeySetup
		return m, m.apiKeyModel.Init()

	case apikey.DoneMsg:
		if msg.Key != "" {
			os.Setenv("ANTHROPIC_API_KEY", msg.Key)
			m.apiKey = msg.Key
		}
		if m.awaitingRecoveryKey && m.sessionID != "" {
			m.awaitingRecoveryKey = false
			m.phase = phaseOperator
			return m, m.setRecoveryKeyAndRetryCmd()
		}
		m.phase = phaseProvisionRuntime
		return m, m.createSessionCmd()

	case sessionCreatedMsg:
		m.sessionID = msg.sessionID
		m.messages = append(m.messages, Message{Role: "system", Content: "Provisioning ap runtime..."})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, m.waitForSessionReadyCmd()

	case sessionReadyMsg:
		m.phase = phaseOperator
		m.messages = append(m.messages, Message{Role: "system", Content: "ap runtime ready. Operator is online."})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, textarea.Blink

	case sessionErrorMsg:
		m.err = msg.err
		return m, nil

	case turnDoneMsg:
		m.submitting = false
		m.pendingRetryContent = ""
		m.messages = append(m.messages, Message{Role: "assistant", Content: msg.content})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case turnErrorMsg:
		m.submitting = false
		if apiErr, ok := asAPIError(msg.err); ok && apiErr.Code == "RECOVERY_KEY_MISSING" {
			m.awaitingRecoveryKey = true
			m.phase = phaseAPIKeySetup
			m.apiKeyModel = apikey.NewModel()
			m.messages = append(m.messages, Message{Role: "system", Content: "Recovery needs your Anthropic API key for this older session. Provide it to continue."})
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, m.apiKeyModel.Init()
		}
		m.err = msg.err
		m.messages = append(m.messages, Message{Role: "system", Content: "Turn failed: " + msg.err.Error()})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	switch m.phase {
	case phaseAPIKeySetup:
		updated, cmd := m.apiKeyModel.Update(msg)
		m.apiKeyModel = updated.(apikey.Model)
		return m, cmd

	case phaseOperator:
		return m.updateOperator(msg)
	}

	return m, nil
}

func (m Model) updateOperator(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.submitting {
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.messages = append(m.messages, Message{Role: "user", Content: text})
			m.input.Reset()
			m.submitting = true
			m.pendingRetryContent = text
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, m.submitTurnCmd(text)
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) createSessionCmd() tea.Cmd {
	apiKey := strings.TrimSpace(m.apiKey)
	if apiKey == "" {
		return func() tea.Msg { return sessionErrorMsg{err: fmt.Errorf("missing ANTHROPIC_API_KEY")} }
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		sess, err := m.api.CreateSession(ctx, controlplane.CreateSessionRequest{
			AnthropicAPIKey: apiKey,
		})
		if err != nil {
			return sessionErrorMsg{err: err}
		}
		return sessionCreatedMsg{sessionID: sess.ID}
	}
}

func (m Model) waitForSessionReadyCmd() tea.Cmd {
	sessionID := m.sessionID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		_, err := m.api.WaitForSessionReady(ctx, sessionID, 3*time.Minute)
		if err != nil {
			return sessionErrorMsg{err: err}
		}
		return sessionReadyMsg{}
	}
}

func (m Model) submitTurnCmd(content string) tea.Cmd {
	sessionID := m.sessionID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		resp, err := m.api.CreateTurn(ctx, sessionID, content)
		if err != nil {
			return turnErrorMsg{err: err}
		}
		if resp.Status != "completed" {
			if resp.Error != "" {
				return turnErrorMsg{err: errors.New(resp.Error)}
			}
			return turnErrorMsg{err: fmt.Errorf("turn status %s", resp.Status)}
		}
		return turnDoneMsg{content: resp.AssistantContent}
	}
}

func (m Model) setRecoveryKeyAndRetryCmd() tea.Cmd {
	sessionID := m.sessionID
	key := strings.TrimSpace(m.apiKey)
	content := strings.TrimSpace(m.pendingRetryContent)
	return func() tea.Msg {
		if key == "" {
			return turnErrorMsg{err: fmt.Errorf("missing ANTHROPIC_API_KEY")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := m.api.SetRecoveryKey(ctx, sessionID, key); err != nil {
			return turnErrorMsg{err: err}
		}
		if content == "" {
			return turnDoneMsg{content: "Recovery key saved. Send your message again."}
		}
		resp, err := m.api.CreateTurn(ctx, sessionID, content)
		if err != nil {
			return turnErrorMsg{err: err}
		}
		if resp.Status != "completed" {
			if resp.Error != "" {
				return turnErrorMsg{err: errors.New(resp.Error)}
			}
			return turnErrorMsg{err: fmt.Errorf("turn status %s", resp.Status)}
		}
		return turnDoneMsg{content: resp.AssistantContent}
	}
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	header := fmt.Sprintf(
		"%s  %s  %s",
		logoStyle.Render("✦ ap"),
		workspaceStyle.Render(m.workspace),
		billingStyle.Render("operator"),
	)
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	switch m.phase {
	case phaseAPIKeySetup:
		b.WriteString(m.apiKeyModel.View())

	case phaseProvisionRuntime:
		b.WriteString(systemMsgStyle.Render("\n  Provisioning ap runtime..."))
		if m.err != nil {
			b.WriteString("\n\n")
			b.WriteString(systemMsgStyle.Render("  Error: " + m.err.Error()))
		}

	case phaseOperator:
		m.viewport.SetContent(m.renderMessages())
		b.WriteString(m.viewport.View())
		b.WriteString("\n")
		b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)))
		b.WriteString("\n")
		b.WriteString(m.input.View())
		b.WriteString("\n")
		help := " enter: send • ctrl+c: quit"
		if m.submitting {
			help = " waiting for ap runtime response (recovery may take a few minutes)... • ctrl+c: quit"
		}
		b.WriteString(helpStyle.Render(help))
		if m.err != nil {
			b.WriteString("\n")
			b.WriteString(systemMsgStyle.Render("  last error: " + m.err.Error()))
		}

	default:
		b.WriteString("\n  Checking API key...")
	}

	return b.String()
}

func asAPIError(err error) (*controlplane.APIError, bool) {
	var apiErr *controlplane.APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return systemMsgStyle.Render("\n  Start an operator conversation. Type a message and press Enter.\n")
	}

	var b strings.Builder
	for _, msg := range m.messages {
		switch msg.Role {
		case "user":
			b.WriteString(inputPromptStyle.Render("  > "))
			b.WriteString(userMsgStyle.Render(msg.Content))
		case "assistant":
			b.WriteString(assistantMsgStyle.Render("  " + msg.Content))
		case "system":
			b.WriteString(systemMsgStyle.Render("  " + msg.Content))
		}
		b.WriteString("\n\n")
	}
	return b.String()
}
