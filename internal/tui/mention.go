package tui

import (
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// maxSuggestions caps how many names the @mention popup offers at once.
const maxSuggestions = 5

// MentionPrefix returns the partial username of an unfinished @mention at the
// end of text, and whether one is in progress. The "@" must start a word, so
// "matt@rdbrck" reads as an address rather than a mention — the same rule
// mentionRegex applies to received messages.
func MentionPrefix(text string) (string, bool) {
	at := strings.LastIndex(text, "@")
	if at < 0 {
		return "", false
	}
	if prev, _ := utf8.DecodeLastRuneInString(text[:at]); isWordRune(prev) {
		return "", false
	}
	prefix := text[at+1:]
	if strings.IndexFunc(prefix, unicode.IsSpace) >= 0 {
		return "", false
	}
	return prefix, true
}

func isWordRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// MatchUsernames returns the candidates starting with prefix, compared
// case-insensitively, sorted alphabetically and capped at maxSuggestions. An
// empty prefix matches everything, so typing a bare "@" lists everyone online.
func MatchUsernames(prefix string, candidates []string) []string {
	lower := strings.ToLower(prefix)
	var matches []string
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToLower(c), lower) {
			matches = append(matches, c)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return strings.ToLower(matches[i]) < strings.ToLower(matches[j])
	})
	if len(matches) > maxSuggestions {
		matches = matches[:maxSuggestions]
	}
	return matches
}

// onlinePeers returns the usernames seen within peerWindow, excluding the
// local user — mentioning yourself never raises a banner.
func (m Model) onlinePeers() []string {
	now := time.Now()
	var peers []string
	for username, info := range m.lastSeen {
		if username != m.username && now.Sub(info.lastSeen) < peerWindow {
			peers = append(peers, username)
		}
	}
	return peers
}

// textBeforeCursor returns the input text left of the cursor. textinput
// indexes its value by rune, so slice by rune too.
func (m Model) textBeforeCursor() string {
	runes := []rune(m.input.Value())
	pos := min(m.input.Position(), len(runes))
	return string(runes[:pos])
}

// refreshSuggestions recomputes the popup from the text left of the cursor.
// Called after every keystroke that reaches the input.
func (m *Model) refreshSuggestions() {
	m.suggestions = nil
	m.suggestionIdx = 0
	if prefix, ok := MentionPrefix(m.textBeforeCursor()); ok {
		m.suggestions = MatchUsernames(prefix, m.onlinePeers())
	}
	m.resizeForSuggestions()
}

// dismissSuggestions closes the popup without completing anything.
func (m *Model) dismissSuggestions() {
	m.suggestions = nil
	m.suggestionIdx = 0
	m.resizeForSuggestions()
}

// resizeForSuggestions borrows the popup's rows from the viewport so the two
// together still fit the terminal, mirroring how showHelp reserves helpHeight.
func (m *Model) resizeForSuggestions() {
	height := 0
	if len(m.suggestions) > 0 {
		height = len(m.suggestions) + 1 // entries plus the header rule
	}
	if height == m.suggestionHeight {
		return
	}
	if m.ready {
		m.viewport.Height += m.suggestionHeight - height
	}
	m.suggestionHeight = height
}

// handleSuggestionKey routes navigation and completion keys to the popup while
// it is open, reporting whether it consumed the key.
func (m *Model) handleSuggestionKey(key string) bool {
	if len(m.suggestions) == 0 {
		return false
	}
	switch key {
	case "up":
		m.suggestionIdx = (m.suggestionIdx - 1 + len(m.suggestions)) % len(m.suggestions)
	case "down":
		m.suggestionIdx = (m.suggestionIdx + 1) % len(m.suggestions)
	case "enter", "tab":
		m.completeMention()
	case "esc":
		m.dismissSuggestions()
	default:
		return false
	}
	return true
}

// completeMention swaps the partial @mention at the cursor for the highlighted
// username and a trailing space, leaving the cursor after it.
func (m *Model) completeMention() {
	runes := []rune(m.input.Value())
	pos := min(m.input.Position(), len(runes))
	before := string(runes[:pos])

	prefix, ok := MentionPrefix(before)
	if !ok {
		return
	}
	completed := before[:len(before)-len(prefix)] + m.suggestions[m.suggestionIdx] + " "

	m.input.SetValue(completed + string(runes[pos:]))
	m.input.SetCursor(len([]rune(completed)))
	m.dismissSuggestions()
}
