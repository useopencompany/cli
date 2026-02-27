package spawn

import "github.com/charmbracelet/lipgloss"

var (
	// Header bar styles.
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	workspaceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true)

	billingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")). // Red.
			Bold(true)

	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")). // Purple.
			Bold(true)

	// Operator area styles.
	userMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	assistantMsgStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("41")) // Green.

	systemMsgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")). // Dim gray.
			Italic(true)

	// Input area.
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99"))

	// Borders and layout.
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
