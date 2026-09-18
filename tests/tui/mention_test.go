package tui_test

import (
	"context"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/esteban/rbchat/internal/network"
	"github.com/esteban/rbchat/internal/tui"
)

func TestMentionPrefix(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		prefix string
		ok     bool
	}{
		{"bare at", "@", "", true},
		{"partial name", "hey @ma", "ma", true},
		{"at start of input", "@ma", "ma", true},
		{"after punctuation", "(@ma", "ma", true},
		{"no at sign", "hello there", "", false},
		{"email address", "mail me at matt@rdbrck", "", false},
		{"completed mention", "@matt hello", "", false},
		{"at followed by space", "hey @ ", "", false},
		{"earlier mention already sent", "@matt hi @es", "es", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prefix, ok := tui.MentionPrefix(tc.text)
			if ok != tc.ok || prefix != tc.prefix {
				t.Fatalf("MentionPrefix(%q) = (%q, %v), want (%q, %v)",
					tc.text, prefix, ok, tc.prefix, tc.ok)
			}
		})
	}
}

func TestMatchUsernames(t *testing.T) {
	candidates := []string{"esteban", "matt", "Marcus", "sofia"}

	cases := []struct {
		name   string
		prefix string
		want   []string
	}{
		{"empty prefix matches all", "", []string{"esteban", "Marcus", "matt", "sofia"}},
		{"case insensitive", "m", []string{"Marcus", "matt"}},
		{"uppercase query", "MA", []string{"Marcus", "matt"}},
		{"narrows to one", "es", []string{"esteban"}},
		{"no match", "zz", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tui.MatchUsernames(tc.prefix, candidates)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("MatchUsernames(%q) = %v, want %v", tc.prefix, got, tc.want)
			}
		})
	}
}

func TestMatchUsernamesCapsResults(t *testing.T) {
	candidates := []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7"}
	if got := tui.MatchUsernames("a", candidates); len(got) != 5 {
		t.Fatalf("expected results capped at 5, got %d", len(got))
	}
}

// modelWithPeers returns a synced model whose peer list holds the given
// usernames, ready to accept typing.
func modelWithPeers(t *testing.T, peers ...string) tea.Model {
	t.Helper()
	database := setupDB(t)
	t.Cleanup(func() { database.Close() })

	// otherInstanceRunning skips the join broadcast, which would need a
	// broadcaster this test does not have.
	var model tea.Model = tui.NewModel(database, "me", "Redbrick", nil, nil, nil,
		context.Background(), func() {}, true, true, "", "1.2.2", "nerd", false)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model, _ = model.Update(tui.SyncTimeoutMsg{})

	for _, name := range peers {
		model, _ = model.Update(tui.IncomingNetworkMsg{Message: network.Message{
			Type:      "heartbeat",
			Username:  name,
			Team:      "Redbrick",
			Text:      "heartbeat",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			MessageID: name + "-hb",
		}})
	}
	return model
}

func typeText(model tea.Model, text string) tea.Model {
	for _, r := range text {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return model
}

func press(model tea.Model, key tea.KeyType) tea.Model {
	model, _ = model.Update(tea.KeyMsg{Type: key})
	return model
}

var ansiCodes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// plainView renders the model with styling stripped, so assertions can match
// on text without tripping over colour escapes.
func plainView(model tea.Model) string {
	return ansiCodes.ReplaceAllString(model.View(), "")
}

func TestMentionPopupListsMatchingPeers(t *testing.T) {
	view := plainView(typeText(modelWithPeers(t, "marcus", "matt", "sofia"), "@m"))

	for _, want := range []string{"@mention", "@marcus", "@matt"} {
		if !strings.Contains(view, want) {
			t.Errorf("expected popup to show %q, view:\n%s", want, view)
		}
	}
	if strings.Contains(view, "@sofia") {
		t.Errorf("expected sofia to be filtered out, view:\n%s", view)
	}
}

func TestMentionPopupExcludesSelf(t *testing.T) {
	view := plainView(typeText(modelWithPeers(t, "me", "matt"), "@m"))

	// Trailing space distinguishes the entry "  @me  Redbrick" from the
	// popup's own "@mention" header.
	if strings.Contains(view, "@me ") {
		t.Errorf("expected own username to be excluded, view:\n%s", view)
	}
}

func TestMentionPopupCompletesOnTab(t *testing.T) {
	model := press(typeText(modelWithPeers(t, "marcus", "matt"), "hey @mar"), tea.KeyTab)

	view := plainView(model)
	if !strings.Contains(view, "hey @marcus ") {
		t.Errorf("expected tab to complete the mention, view:\n%s", view)
	}
	if strings.Contains(view, "@mention") {
		t.Errorf("expected popup to close after completing, view:\n%s", view)
	}
}

func TestMentionPopupCompletesOnEnter(t *testing.T) {
	model := press(typeText(modelWithPeers(t, "marcus", "matt"), "hey @mar"), tea.KeyEnter)

	if view := plainView(model); !strings.Contains(view, "hey @marcus ") {
		t.Errorf("expected enter to complete rather than send, view:\n%s", view)
	}
}

func TestMentionPopupNavigatesWithArrowKeys(t *testing.T) {
	// Sorted case-insensitively, "marcus" leads and "matt" follows.
	model := typeText(modelWithPeers(t, "marcus", "matt"), "@m")
	model = press(press(model, tea.KeyDown), tea.KeyEnter)

	if view := plainView(model); !strings.Contains(view, "@matt ") {
		t.Errorf("expected down arrow to move to the second entry, view:\n%s", view)
	}
}

func TestMentionPopupArrowNavigationWraps(t *testing.T) {
	// Up from the first entry wraps to the last.
	model := typeText(modelWithPeers(t, "marcus", "matt"), "@m")
	model = press(press(model, tea.KeyUp), tea.KeyEnter)

	if view := plainView(model); !strings.Contains(view, "@matt ") {
		t.Errorf("expected up arrow to wrap to the last entry, view:\n%s", view)
	}
}

func TestMentionPopupDismissedByEsc(t *testing.T) {
	model := press(typeText(modelWithPeers(t, "marcus", "matt"), "@m"), tea.KeyEsc)

	view := plainView(model)
	if strings.Contains(view, "@mention") {
		t.Errorf("expected esc to dismiss the popup, view:\n%s", view)
	}
	if !strings.Contains(view, "@m") {
		t.Errorf("expected esc to leave typed text alone, view:\n%s", view)
	}
}

func TestMentionPopupIgnoresEmailAddresses(t *testing.T) {
	view := plainView(typeText(modelWithPeers(t, "marcus", "matt"), "reach me at matt@ma"))

	if strings.Contains(view, "@mention") {
		t.Errorf("expected no popup mid-word, view:\n%s", view)
	}
}

func TestMentionPopupStaysClosedWithoutMatches(t *testing.T) {
	view := plainView(typeText(modelWithPeers(t, "marcus", "matt"), "@zz"))

	if strings.Contains(view, "@mention") {
		t.Errorf("expected no popup when nothing matches, view:\n%s", view)
	}
}

func lineCount(model tea.Model) int {
	return strings.Count(model.View(), "\n") + 1
}

// The popup borrows its rows from the viewport, so opening, resizing, and
// closing it must leave the overall height untouched.
func TestMentionPopupKeepsViewHeightStable(t *testing.T) {
	model := modelWithPeers(t, "marcus", "matt", "mehrad", "sofia")
	closed := lineCount(model)

	steps := []struct {
		name string
		act  func(tea.Model) tea.Model
	}{
		{"one match", func(m tea.Model) tea.Model { return typeText(m, "@mar") }},
		{"widen to three", func(m tea.Model) tea.Model { return press(press(m, tea.KeyBackspace), tea.KeyBackspace) }},
		{"widen to all", func(m tea.Model) tea.Model { return press(m, tea.KeyBackspace) }},
		{"complete", func(m tea.Model) tea.Model { return press(m, tea.KeyEnter) }},
		{"reopen", func(m tea.Model) tea.Model { return typeText(m, "@m") }},
		{"dismiss", func(m tea.Model) tea.Model { return press(m, tea.KeyEsc) }},
	}
	for _, step := range steps {
		model = step.act(model)
		if got := lineCount(model); got != closed {
			t.Errorf("after %q: view is %d lines, want %d", step.name, got, closed)
		}
	}
}
