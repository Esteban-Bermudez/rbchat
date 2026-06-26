package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/esteban/rbchat/internal/network"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	msgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	usernameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true)

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	syncStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Bold(true)
)

func (m Model) View() string {
	if m.quitting {
		return m.err.Error() + "\n"
	}

	if m.syncing {
		return syncStyle.Render(" Syncing with the mesh... ") + "\n\n" +
			"Waiting for peers to respond...\n" +
			"(continuing in a moment)\n"
	}

	if !m.ready {
		return "\n  Loading...\n"
	}

	title := titleStyle.Render(fmt.Sprintf(" rbchat | %s | %d peers online ", multicastAddr, m.peerCount))
	title += "\n"

	separator := strings.Repeat("─", m.viewport.Width)
	title += separator + "\n"

	chatContent := m.viewport.View()

	var inputField string
	if m.err != nil {
		inputField = errorStyle.Render(fmt.Sprintf("⚠ %v", m.err)) + "\n"
	} else {
		inputField += "\n"
	}
	inputField += m.input.View()

	return title + chatContent + "\n" + inputField
}

func renderMessage(msg network.Message) string {
	switch msg.Type {
	case "join":
		t := parseTimestamp(msg.Timestamp)
		return systemStyle.Render(fmt.Sprintf("[%s] %s (%s) %s",
			t, msg.Username, msg.Team, msg.Text))

	case "chat":
		t := parseTimestamp(msg.Timestamp)
		ts := timestampStyle.Render("[" + t + "]")
		user := usernameStyle.Render(msg.Username)
		var teamPart string
		if msg.Team != "" {
			teamPart = " (" + msg.Team + ")"
		}
		return msgStyle.Render(fmt.Sprintf("%s %s%s: %s", ts, user, teamPart, msg.Text))

	case "sync":
		return ""

	default:
		return msgStyle.Render(msg.Text)
	}
}

func parseTimestamp(ts string) string {
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return parsed.Format("Jan 2 15:04")
}
