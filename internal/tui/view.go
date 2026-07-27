package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/esteban/rbchat/internal/network"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#1A1B26")).
			Padding(0, 1)

	msgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E0E0E0"))

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A0A0A0")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444"))

	timestampStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	syncStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	bellOnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")).
			Background(lipgloss.Color("#1A1B26"))

	bellOffStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#EF4444")).
			Background(lipgloss.Color("#1A1B26"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280"))

	mentionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#EF4444"))

	insecureStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#F59E0B"))
)

var (
	nerdIcons = map[string]string{
		"darwin":  "\U000f0179",
		"windows": "\U000f017a",
		"linux":   "\U000f017c",
	}

	txtIcons = map[string]string{
		"darwin":  "[mac]",
		"windows": "[win]",
		"linux":   "[lnx]",
	}

	emojiIcons = map[string]string{
		"darwin":  "\U0001f34e",
		"windows": "\U0001fa9f",
		"linux":   "\U0001f427",
	}
)

func osIcon(os, mode string) string {
	if os == "" {
		return ""
	}
	var m map[string]string
	switch mode {
	case "nerd":
		m = nerdIcons
	case "text":
		m = txtIcons
	case "emoji":
		m = emojiIcons
	default:
		return ""
	}
	return m[os]
}

func rightSide(os, mode, ts string) string {
	icon := osIcon(os, mode)
	if icon == "" {
		return ts
	}
	return timestampStyle.Render(icon) + " " + ts
}

var teamColors = map[string]lipgloss.Color{
	"Animoto":   "#FFD700",
	"Delivra":   "#00CED1",
	"Duplex":    "#FFA500",
	"Leadpages": "#7C3AED",
	"Paved":     "#10B981",
	"Shift":     "#3B82F6",
	"Redbrick":  "#EF4444",
}

func teamStyle(team string) lipgloss.Style {
	c, ok := teamColors[team]
	if !ok {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(c)
}

func (m Model) View() string {
	if m.quitting {
		return m.err.Error() + "\n"
	}

	if m.syncing {
		str := syncStyle.Render(" Syncing with the mesh... ") +
			"\n\n" +
			"Waiting for peers to respond...\n" +
			"(continuing in a moment)\n"

		// Place in the center of the viewport
		return lipgloss.Place(
			m.viewport.Width,
			m.viewport.Height,
			lipgloss.Center,
			lipgloss.Center,
			str,
		)
	}

	if !m.ready {
		return "\n  Loading...\n"
	}

	var bell string
	switch m.notificationMode {
	case NotifyAll:
		bell = bellOnStyle.Render("🔔")
	case NotifyMentions:
		bell = bellOnStyle.Render("🔔@")
	default:
		bell = bellOffStyle.Render("🔕")
	}
	var versionStr string
	if m.updateAvailable != "" {
		versionStr = fmt.Sprintf("⬆ %s — rbchat update", m.updateAvailable)
	} else if m.version == "" || m.version == "dev" {
		versionStr = "dev"
	} else {
		versionStr = "v" + m.version
	}
	title := titleStyle.Render(fmt.Sprintf(" rbchat | %s | ", multicastAddr)) +
		bell +
		titleStyle.Render(fmt.Sprintf(" | %d peers ", m.peerCount)) +
		titleStyle.Render(" | ? for help ") +
		titleStyle.Render(fmt.Sprintf(" | %s ", versionStr))
	title += "\n"

	separator := strings.Repeat("─", m.viewport.Width)
	title += separator + "\n"

	var chatContent string = m.viewport.View()

	var inputField string
	if m.err != nil {
		inputField = errorStyle.Render(fmt.Sprintf("⚠ %v", m.err)) + "\n"
	}
	inputField += m.input.View()
	if m.showHelp {
		inputField += "\n" + helpPanel(m.viewport.Width)
	}

	out := title + chatContent + "\n" + inputField
	if banner := m.mentionBanner(); banner != "" {
		out = banner + "\n" + out
	}
	if m.signingDisabled {
		text := " ⚠ Message signing is disabled — set RBCHAT_INSECURE=1 only for local development "
		style := insecureStyle
		if m.viewport.Width > 0 {
			style = style.Width(m.viewport.Width)
		}
		out = style.Render(text) + "\n" + out
	}
	return out
}

// mentionBanner renders the "@ mentioned you" line shown at the top of the
// screen while a mention is active, or "" when none is.
func (m Model) mentionBanner() string {
	if m.mentionBy == "" {
		return ""
	}
	text := fmt.Sprintf(" 🔔 %s mentioned you in a message ", m.mentionBy)
	style := mentionStyle
	if m.viewport.Width > 0 {
		style = style.Width(m.viewport.Width)
	}
	return style.Render(text)
}

// highlightMentions wraps occurrences of @username in mentionStyle.
func highlightMentions(text, username string) string {
	if username == "" {
		return msgStyle.Render(text)
	}
	re := mentionRegex(username)
	var result strings.Builder
	lastEnd := 0
	for _, m := range re.FindAllStringIndex(text, -1) {
		if m[0] > lastEnd {
			result.WriteString(msgStyle.Render(text[lastEnd:m[0]]))
		}
		result.WriteString(mentionStyle.Render(text[m[0]:m[1]]))
		lastEnd = m[1]
	}
	if lastEnd < len(text) {
		result.WriteString(msgStyle.Render(text[lastEnd:]))
	}
	if result.Len() == 0 {
		return msgStyle.Render(text)
	}
	return result.String()
}

func RenderMessage(msg network.Message) string {
	return renderMessage(msg, 0, "", "")
}

func renderMessage(msg network.Message, width int, osIconMode string, highlightUsername string) string {
	switch msg.Type {
	case "join":
		t := parseTimestamp(msg.Timestamp)
		ts := timestampStyle.Render("[" + t + "]")
		user := msg.Username
		var teamPart string
		if msg.Team != "" {
			teamPart = teamStyle(msg.Team).Render(" (" + msg.Team + ")")
		}
		text := systemStyle.Render(" " + msg.Text)
		left := fmt.Sprintf("%s%s%s", user, teamPart, text)
		right := rightSide(msg.OS, osIconMode, ts)
		if width > 0 {
			leftWidth := lipgloss.Width(left)
			rightWidth := lipgloss.Width(right)
			pad := width - leftWidth - rightWidth
			if pad < 1 {
				pad = 1
			}
			return left + strings.Repeat(" ", pad) + right
		}
		return fmt.Sprintf("%s %s", left, right)

	case "chat":
		t := parseTimestamp(msg.Timestamp)
		ts := timestampStyle.Render("[" + t + "]")
		user := msg.Username
		var teamPart string
		if msg.Team != "" {
			teamPart = teamStyle(msg.Team).Render(" (" + msg.Team + ")")
		}
		header := fmt.Sprintf("%s%s:", user, teamPart)
		left := fmt.Sprintf("%s %s", header, highlightMentions(msg.Text, highlightUsername))
		right := rightSide(msg.OS, osIconMode, ts)
		if width > 0 {
			leftWidth := lipgloss.Width(left)
			rightWidth := lipgloss.Width(right)
			pad := width - leftWidth - rightWidth
			if pad < 1 {
				pad = 1
			}
			return left + strings.Repeat(" ", pad) + right
		}
		return fmt.Sprintf("%s %s", left, right)

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

func messageDate(ts string) string {
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return parsed.Format("Jan 2, 2006")
}

func helpPanel(width int) string {
	if width <= 0 {
		width = 60
	}
	// Build header as "─ Keybindings ──────" without byte-splicing
	// styled strings (which breaks ANSI escape codes).
	headerText := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FBBF24")).
		Render(" Keybindings ")
	headerWidth := lipgloss.Width(headerText)
	pad := width - headerWidth
	if pad < 2 {
		pad = 2
	}
	header := dividerStyle.Render("─") + headerText + dividerStyle.Render(strings.Repeat("─", pad-1))

	items := []struct{ key, desc string }{
		{"ctrl+n", "Cycle notifications (all/@mentions/off)"},
		{"ctrl+t", "Toggle online users list"},
		{"enter", "Send message"},
		{"ctrl+u", "Scroll up (half page)"},
		{"ctrl+d", "Scroll down (half page)"},
		{"pgup", "Page up"},
		{"pgdown", "Page down"},
		{"esc", "Dismiss @mention banner"},
		{"ctrl+c", "Quit"},
		{"?", "Close this help"},
	}
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9CA3AF"))

	var body string
	for _, it := range items {
		body += fmt.Sprintf("  %s  %s\n",
			keyStyle.Render(fmt.Sprintf("%-8s", it.key)),
			descStyle.Render(it.desc),
		)
	}

	return header + "\n" + body + dividerStyle.Render(strings.Repeat("─", width))
}

// buildUserListContent renders the online users list as viewport content
// (same format as the chat log).
func buildUserListContent(width int, lastSeen map[string]peerInfo, osIconMode string) string {
	now := time.Now()
	type entry struct {
		username, team string
	}
	var users []entry
	for username, info := range lastSeen {
		if now.Sub(info.lastSeen) < peerWindow {
			users = append(users, entry{username, info.team})
		}
	}
	sort.Slice(users, func(i, j int) bool {
		return strings.ToLower(users[i].username) < strings.ToLower(users[j].username)
	})

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FBBF24")).
		Render(fmt.Sprintf("   Online Users (%d)", len(users)))

	var sb strings.Builder
	sb.WriteString(header + "\n\n")
	for _, u := range users {
		teamPart := ""
		if u.team != "" {
			teamPart = teamStyle(u.team).Render(" " + u.team)
		}
		sb.WriteString(fmt.Sprintf("  %s%s\n", u.username, teamPart))
	}
	sb.WriteString("\n" + dividerStyle.Render(strings.Repeat("─", width)))

	return sb.String()
}

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			result.WriteByte('\n')
		}
		remaining := line
		for len(remaining) > width {
			idx := strings.LastIndex(remaining[:width+1], " ")
			if idx <= 0 {
				idx = width
			}
			result.WriteString(remaining[:idx])
			result.WriteByte('\n')
			remaining = strings.TrimLeft(remaining[idx:], " ")
		}
		result.WriteString(remaining)
	}
	return result.String()
}
