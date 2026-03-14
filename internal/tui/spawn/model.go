package spawn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"go.agentprotocol.cloud/cli/internal/controlplane"
)

// phase tracks where we are in the spawn lifecycle.
type phase int

const (
	phaseProvisionRuntime phase = iota
	phaseOperator
)

// Message represents a single session message.
type Message struct {
	ID      string
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
	activeRunID         string
	lastRunSequence     int
	runEvents           <-chan controlplane.RunEvent
	runErrors           <-chan error
	streamCancel        context.CancelFunc
	streamingMessageID  string

	// Session state.
	messages   []Message
	viewport   viewport.Model
	input      textarea.Model
	err        error
	submitting bool
}

// NewModel creates a spawn TUI for creating a new session.
func NewModel(workspace string, client *controlplane.Client) Model {
	return newModel(workspace, client, "", nil)
}

// NewResumeModel creates a spawn TUI for an existing session.
func NewResumeModel(workspace string, client *controlplane.Client, sessionID string, history []controlplane.Message) Model {
	messages := make([]Message, 0, len(history))
	for _, msg := range history {
		messages = append(messages, Message{ID: msg.ID, Role: msg.Role, Content: msg.Content})
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

	startPhase := phaseProvisionRuntime
	if sessionID != "" {
		startPhase = phaseOperator
	}

	return Model{
		workspace: workspace,
		phase:     startPhase,
		api:       client,
		sessionID: sessionID,
		messages:  messages,
		viewport:  vp,
		input:     ta,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	switch m.phase {
	case phaseProvisionRuntime:
		return m.createSessionCmd()
	case phaseOperator:
		return textarea.Blink
	default:
		return nil
	}
}

type sessionCreatedMsg struct{ session *controlplane.Session }
type sessionErrorMsg struct{ err error }
type runCreatedMsg struct{ run *controlplane.Run }
type runEventMsg struct{ event controlplane.RunEvent }
type runStreamClosedMsg struct{}
type runErrorMsg struct{ err error }

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
		return m, nil

	case sessionCreatedMsg:
		if msg.session != nil {
			m.sessionID = msg.session.ID
		}
		m.phase = phaseOperator
		readyMsg := "Session is online."
		m.messages = append(m.messages, Message{Role: "system", Content: readyMsg})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, textarea.Blink

	case sessionErrorMsg:
		m.err = msg.err
		return m, nil

	case runCreatedMsg:
		if msg.run == nil {
			return m, nil
		}
		m.activeRunID = msg.run.ID
		m.lastRunSequence = 0
		if m.streamCancel != nil {
			m.streamCancel()
		}
		streamCtx, cancel := context.WithCancel(context.Background())
		m.streamCancel = cancel
		m.runEvents, m.runErrors = m.api.StreamRunEvents(streamCtx, msg.run.ID, 0)
		return m, waitForRunEventCmd(m.runEvents, m.runErrors)

	case runEventMsg:
		if msg.event.RunID == "" || msg.event.RunID != m.activeRunID {
			return m, nil
		}
		if msg.event.SequenceNumber > m.lastRunSequence {
			m.lastRunSequence = msg.event.SequenceNumber
		}
		m.applyRunEvent(msg.event)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		if isTerminalRunEvent(msg.event.Type) {
			m.submitting = false
			m.activeRunID = ""
			m.streamingMessageID = ""
			if m.streamCancel != nil {
				m.streamCancel()
				m.streamCancel = nil
			}
			return m, nil
		}
		return m, waitForRunEventCmd(m.runEvents, m.runErrors)

	case runStreamClosedMsg:
		return m, nil

	case runErrorMsg:
		m.submitting = false
		m.activeRunID = ""
		m.streamingMessageID = ""
		if m.streamCancel != nil {
			m.streamCancel()
			m.streamCancel = nil
		}
		m.err = msg.err
		m.messages = append(m.messages, Message{Role: "system", Content: "Run failed: " + msg.err.Error()})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil
	}

	switch m.phase {
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
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		sess, err := m.api.CreateSession(ctx, controlplane.CreateSessionRequest{
			Source: "cli",
		})
		if err != nil {
			return sessionErrorMsg{err: err}
		}
		return sessionCreatedMsg{session: sess}
	}
}

func (m Model) submitTurnCmd(content string) tea.Cmd {
	sessionID := m.sessionID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		resp, err := m.api.CreateRun(ctx, sessionID, controlplane.CreateRunRequest{
			Content: content,
		})
		if err != nil {
			return runErrorMsg{err: err}
		}
		return runCreatedMsg{run: resp}
	}
}

func waitForRunEventCmd(events <-chan controlplane.RunEvent, errs <-chan error) tea.Cmd {
	return func() tea.Msg {
		select {
		case event, ok := <-events:
			if !ok {
				return runStreamClosedMsg{}
			}
			return runEventMsg{event: event}
		case err, ok := <-errs:
			if !ok || err == nil {
				return runStreamClosedMsg{}
			}
			return runErrorMsg{err: err}
		}
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
	case phaseProvisionRuntime:
		b.WriteString(systemMsgStyle.Render("\n  Starting session..."))
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
			help = " waiting for session response... • ctrl+c: quit"
		}
		b.WriteString(helpStyle.Render(help))
		if m.err != nil {
			b.WriteString("\n")
			b.WriteString(systemMsgStyle.Render("  last error: " + m.err.Error()))
		}

	default:
		b.WriteString("\n  Starting session...")
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
	return "session"
}

func isTerminalRunEvent(eventType string) bool {
	switch eventType {
	case "run.completed", "run.failed", "run.cancelled":
		return true
	default:
		return false
	}
}

func (m Model) renderMessages() string {
	if len(m.messages) == 0 {
		return systemMsgStyle.Render("\n  Start a session. Type a message and press Enter.\n")
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

func (m *Model) applyRunEvent(event controlplane.RunEvent) {
	switch event.Type {
	case "run.created":
		m.messages = append(m.messages, Message{ID: event.ID, Role: "system", Content: "Run queued."})
	case "run.started":
		m.messages = append(m.messages, Message{ID: event.ID, Role: "system", Content: "Run started."})
	case "reasoning.summary.delta":
		if delta, ok := event.Data["delta"].(string); ok && strings.TrimSpace(delta) != "" {
			m.messages = append(m.messages, Message{ID: event.ID, Role: "tool", Content: delta})
		}
	case "tool_call.started":
		if name, ok := event.Data["name"].(string); ok && strings.TrimSpace(name) != "" {
			m.messages = append(m.messages, Message{ID: event.ID, Role: "tool", Content: "Tool started: " + name})
		}
	case "tool_call.completed":
		if name, ok := event.Data["name"].(string); ok && strings.TrimSpace(name) != "" {
			m.messages = append(m.messages, Message{ID: event.ID, Role: "tool", Content: "Tool completed: " + name})
		}
	case "tool_call.failed":
		name, _ := event.Data["name"].(string)
		errText, _ := event.Data["error"].(string)
		content := "Tool failed"
		if strings.TrimSpace(name) != "" {
			content = "Tool failed: " + name
		}
		if strings.TrimSpace(errText) != "" {
			content += " (" + errText + ")"
		}
		m.messages = append(m.messages, Message{ID: event.ID, Role: "tool", Content: content})
	case "message.output.delta":
		delta, _ := event.Data["delta"].(string)
		if strings.TrimSpace(delta) == "" {
			return
		}
		if m.streamingMessageID == "" {
			m.streamingMessageID = event.ID
			m.messages = append(m.messages, Message{ID: event.ID, Role: "assistant", Content: delta})
			return
		}
		for i := range m.messages {
			if m.messages[i].ID == m.streamingMessageID {
				m.messages[i].Content += delta
				return
			}
		}
		m.messages = append(m.messages, Message{ID: m.streamingMessageID, Role: "assistant", Content: delta})
	case "message.output.completed":
		if text, ok := event.Data["text"].(string); ok && strings.TrimSpace(text) != "" {
			if m.streamingMessageID != "" {
				for i := range m.messages {
					if m.messages[i].ID == m.streamingMessageID {
						m.messages[i].Content = text
						break
					}
				}
			} else {
				m.messages = append(m.messages, Message{ID: event.ID, Role: "assistant", Content: text})
			}
		}
		m.streamingMessageID = ""
	case "run.completed":
		m.messages = append(m.messages, Message{ID: event.ID, Role: "system", Content: "Run completed."})
	case "run.failed":
		errText, _ := event.Data["error"].(string)
		if errText == "" {
			errText = "run failed"
		}
		m.err = errors.New(errText)
		m.messages = append(m.messages, Message{ID: event.ID, Role: "system", Content: "Run failed: " + errText})
	case "run.cancelled":
		m.messages = append(m.messages, Message{ID: event.ID, Role: "system", Content: "Run cancelled."})
	}
}
