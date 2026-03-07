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
	ID      string
	Role    string
	Content string
}

// Model is the top-level TUI model for the spawn command.
type Model struct {
	workspace       string
	phase           phase
	width           int
	height          int
	keyVaultDocsURL string
	agentID         string
	agentVersion    string

	api                 *controlplane.Client
	sessionID           string
	activeTurnID        string
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
func NewModel(workspace string, client *controlplane.Client, keyVaultDocsURL, agentID string) Model {
	return newModel(workspace, client, "", nil, keyVaultDocsURL, agentID, "")
}

// NewResumeModel creates a spawn TUI for an existing operator session.
func NewResumeModel(workspace string, client *controlplane.Client, sessionID string, history []controlplane.Message, keyVaultDocsURL, agentID, agentVersion string) Model {
	messages := make([]Message, 0, len(history))
	for _, msg := range history {
		messages = append(messages, Message{ID: msg.ID, Role: msg.Role, Content: msg.Content})
	}
	return newModel(workspace, client, sessionID, messages, keyVaultDocsURL, agentID, agentVersion)
}

func newModel(workspace string, client *controlplane.Client, sessionID string, messages []Message, keyVaultDocsURL, agentID, agentVersion string) Model {
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
	if envKey := os.Getenv("ANTHROPIC_API_KEY"); envKey != "" {
		apiKey = envKey
	}
	if sessionID != "" {
		startPhase = phaseOperator
	} else if apiKey != "" {
		startPhase = phaseProvisionRuntime
	}

	return Model{
		workspace:       workspace,
		phase:           startPhase,
		keyVaultDocsURL: keyVaultDocsURL,
		agentID:         strings.TrimSpace(agentID),
		agentVersion:    strings.TrimSpace(agentVersion),
		api:             client,
		sessionID:       sessionID,
		apiKey:          apiKey,
		apiKeyModel:     apikey.NewModel(keyVaultDocsURL),
		messages:        messages,
		viewport:        vp,
		input:           ta,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	switch m.phase {
	case phaseAPIKeyCheck:
		return m.createSessionCmd(true)
	case phaseProvisionRuntime:
		return m.createSessionCmd(false)
	case phaseOperator:
		return textarea.Blink
	default:
		return nil
	}
}

type apiKeyMissingMsg struct{}
type sessionCreatedMsg struct{ session *controlplane.Session }
type sessionReadyMsg struct{}
type sessionErrorMsg struct{ err error }
type turnCreatedMsg struct{ resp *controlplane.TurnResponse }
type turnPolledMsg struct{ resp *controlplane.TurnResponse }
type pollTurnMsg struct{ turnID string }
type turnDoneMsg struct{ content string }
type turnErrorMsg struct{ err error }
type apiKeyVaultSavedMsg struct{ err error }

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
		m.apiKeyModel = apikey.NewModel(m.keyVaultDocsURL)
		return m, m.apiKeyModel.Init()

	case apikey.DoneMsg:
		if msg.Key != "" {
			m.apiKey = msg.Key
		}
		if msg.InVault {
			return m, m.saveAPIKeyToVaultCmd()
		}
		if msg.Key != "" {
			os.Setenv("ANTHROPIC_API_KEY", msg.Key)
		}
		return m.advanceAfterAPIKeyReady()

	case apiKeyVaultSavedMsg:
		if msg.err != nil {
			m.phase = phaseAPIKeySetup
			m.apiKeyModel = apikey.NewStorageModel(m.apiKey, m.keyVaultDocsURL, msg.err.Error())
			return m, nil
		}
		return m.advanceAfterAPIKeyReady()

	case sessionCreatedMsg:
		if msg.session != nil {
			m.sessionID = msg.session.ID
			if strings.TrimSpace(msg.session.AgentID) != "" {
				m.agentID = strings.TrimSpace(msg.session.AgentID)
			}
			if strings.TrimSpace(msg.session.AgentVersion) != "" {
				m.agentVersion = strings.TrimSpace(msg.session.AgentVersion)
			}
		}
		m.messages = append(m.messages, Message{Role: "system", Content: "Provisioning ap runtime..."})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, m.waitForSessionReadyCmd()

	case sessionReadyMsg:
		m.phase = phaseOperator
		readyMsg := "ap runtime ready. Operator is online."
		if strings.TrimSpace(m.agentID) != "" {
			readyMsg = "ap runtime ready. " + m.agentID + " is online."
		}
		m.messages = append(m.messages, Message{Role: "system", Content: readyMsg})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, textarea.Blink

	case sessionErrorMsg:
		m.err = msg.err
		return m, nil

	case turnDoneMsg:
		m.submitting = false
		m.activeTurnID = ""
		m.pendingRetryContent = ""
		m.messages = append(m.messages, Message{Role: "assistant", Content: msg.content})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case turnCreatedMsg:
		m.activeTurnID = msg.resp.TurnID
		m.mergeMessages(msg.resp.Messages)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, m.scheduleTurnPollCmd(msg.resp.TurnID)

	case pollTurnMsg:
		if msg.turnID == "" || msg.turnID != m.activeTurnID {
			return m, nil
		}
		return m, m.fetchTurnCmd(msg.turnID)

	case turnPolledMsg:
		m.mergeMessages(msg.resp.Messages)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		switch msg.resp.Status {
		case "completed":
			m.submitting = false
			m.activeTurnID = ""
			m.pendingRetryContent = ""
			return m, nil
		case "failed":
			m.submitting = false
			m.activeTurnID = ""
			if msg.resp.Code == "RECOVERY_KEY_MISSING" {
				m.awaitingRecoveryKey = true
				if strings.TrimSpace(m.apiKey) != "" {
					return m, m.setRecoveryKeyAndRetryCmd(false)
				}
				m.messages = append(m.messages, Message{Role: "system", Content: "Recovery needs your Anthropic API key for this older session. Checking ap key vault first."})
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
				return m, m.setRecoveryKeyAndRetryCmd(true)
			}
			m.err = errors.New(msg.resp.Error)
			m.messages = append(m.messages, Message{Role: "system", Content: "Turn failed: " + msg.resp.Error})
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, nil
		default:
			return m, m.scheduleTurnPollCmd(msg.resp.TurnID)
		}

	case turnErrorMsg:
		m.submitting = false
		m.activeTurnID = ""
		if apiErr, ok := asAPIError(msg.err); ok && apiErr.Code == "RECOVERY_KEY_MISSING" {
			m.awaitingRecoveryKey = true
			if strings.TrimSpace(m.apiKey) != "" {
				return m, m.setRecoveryKeyAndRetryCmd(false)
			}
			m.messages = append(m.messages, Message{Role: "system", Content: "Recovery needs your Anthropic API key for this older session. Checking ap key vault first."})
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
			return m, m.setRecoveryKeyAndRetryCmd(true)
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

func (m Model) advanceAfterAPIKeyReady() (tea.Model, tea.Cmd) {
	if m.awaitingRecoveryKey && m.sessionID != "" {
		m.awaitingRecoveryKey = false
		m.phase = phaseOperator
		return m, m.setRecoveryKeyAndRetryCmd(false)
	}
	m.phase = phaseProvisionRuntime
	return m, m.createSessionCmd(false)
}

func (m Model) createSessionCmd(useAPKeyVault bool) tea.Cmd {
	apiKey := strings.TrimSpace(m.apiKey)
	if apiKey == "" && !useAPKeyVault {
		return func() tea.Msg { return sessionErrorMsg{err: fmt.Errorf("missing ANTHROPIC_API_KEY")} }
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		sess, err := m.api.CreateSession(ctx, controlplane.CreateSessionRequest{
			AnthropicAPIKey: apiKey,
			UseAPKeyVault:   useAPKeyVault,
			AgentID:         strings.TrimSpace(m.agentID),
		})
		if err != nil {
			if useAPKeyVault && isKeyVaultPromptError(err) {
				return apiKeyMissingMsg{}
			}
			return sessionErrorMsg{err: err}
		}
		return sessionCreatedMsg{session: sess}
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

		wait := false
		resp, err := m.api.CreateTurn(ctx, sessionID, controlplane.CreateTurnRequest{
			Content: content,
			Wait:    &wait,
		})
		if err != nil {
			return turnErrorMsg{err: err}
		}
		return turnCreatedMsg{resp: resp}
	}
}

func (m Model) saveAPIKeyToVaultCmd() tea.Cmd {
	key := strings.TrimSpace(m.apiKey)
	return func() tea.Msg {
		if key == "" {
			return apiKeyVaultSavedMsg{err: fmt.Errorf("missing ANTHROPIC_API_KEY")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		return apiKeyVaultSavedMsg{err: m.api.SaveAnthropicKeyToVault(ctx, key)}
	}
}

func (m Model) setRecoveryKeyAndRetryCmd(useAPKeyVault bool) tea.Cmd {
	sessionID := m.sessionID
	key := strings.TrimSpace(m.apiKey)
	content := strings.TrimSpace(m.pendingRetryContent)
	return func() tea.Msg {
		if key == "" && !useAPKeyVault {
			return turnErrorMsg{err: fmt.Errorf("missing ANTHROPIC_API_KEY")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := m.api.SetRecoveryKey(ctx, sessionID, key, useAPKeyVault); err != nil {
			if useAPKeyVault && isKeyVaultPromptError(err) {
				return apiKeyMissingMsg{}
			}
			return turnErrorMsg{err: err}
		}
		if content == "" {
			return turnDoneMsg{content: "Recovery key saved. Send your message again."}
		}
		wait := false
		resp, err := m.api.CreateTurn(ctx, sessionID, controlplane.CreateTurnRequest{
			Content: content,
			Wait:    &wait,
		})
		if err != nil {
			return turnErrorMsg{err: err}
		}
		return turnCreatedMsg{resp: resp}
	}
}

func (m Model) scheduleTurnPollCmd(turnID string) tea.Cmd {
	return tea.Tick(350*time.Millisecond, func(time.Time) tea.Msg {
		return pollTurnMsg{turnID: turnID}
	})
}

func (m Model) fetchTurnCmd(turnID string) tea.Cmd {
	sessionID := m.sessionID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := m.api.GetTurn(ctx, sessionID, turnID)
		if err != nil {
			return turnErrorMsg{err: err}
		}
		return turnPolledMsg{resp: resp}
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
		billingStyle.Render(m.modeLabel()),
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
		b.WriteString("\n  Checking ap key vault...")
		if m.err != nil {
			b.WriteString("\n\n")
			b.WriteString(systemMsgStyle.Render("  Error: " + m.err.Error()))
		}
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

func (m Model) modeLabel() string {
	if strings.TrimSpace(m.agentID) == "" {
		return "operator"
	}
	if strings.TrimSpace(m.agentVersion) == "" {
		return "agent: " + m.agentID
	}
	return "agent: " + m.agentID + "@" + m.agentVersion
}

func isKeyVaultPromptError(err error) bool {
	apiErr, ok := asAPIError(err)
	if !ok {
		return false
	}
	return apiErr.Code == "KEY_VAULT_EMPTY" || apiErr.Code == "KEY_VAULT_INVALID"
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
		case "tool":
			b.WriteString(toolMsgStyle.Render("  " + msg.Content))
		case "system":
			b.WriteString(systemMsgStyle.Render("  " + msg.Content))
		}
		b.WriteString("\n\n")
	}
	return b.String()
}

func (m *Model) mergeMessages(incoming []controlplane.Message) {
	for _, msg := range incoming {
		if msg.ID == "" {
			continue
		}
		found := false
		for i := range m.messages {
			if m.messages[i].ID != msg.ID {
				continue
			}
			m.messages[i].Role = msg.Role
			m.messages[i].Content = msg.Content
			found = true
			break
		}
		if !found {
			m.messages = append(m.messages, Message{
				ID:      msg.ID,
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
}
