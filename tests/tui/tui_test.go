package tui_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/esteban/rbchat/internal/db"
	"github.com/esteban/rbchat/internal/network"
	"github.com/esteban/rbchat/internal/tui"
	_ "modernc.org/sqlite"
)

func TestTeamsNonEmpty(t *testing.T) {
	teams := tui.GetTeams()
	if len(teams) == 0 {
		t.Fatal("expected at least one team")
	}
}

func TestTitleShowsHelpHint(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	model := tui.NewModel(database, "me", "Redbrick", nil, nil, nil, ctx, func() {}, true, true, "", "1.2.2", "nerd")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	updated, _ = updated.Update(tui.SyncTimeoutMsg{})

	if view := updated.View(); !strings.Contains(view, "? for help") {
		t.Fatalf("expected title to show help hint, view: %q", view)
	}
}

func TestRenderMessageChat(t *testing.T) {
	msg := network.Message{
		Type:      "chat",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "hello",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "abc",
	}
	rendered := tui.RenderMessage(msg)
	if rendered == "" {
		t.Fatal("expected non-empty render")
	}
}

func TestRenderMessageSync(t *testing.T) {
	msg := network.Message{
		Type:      "sync",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "hello",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "abc",
	}
	rendered := tui.RenderMessage(msg)
	if rendered != "" {
		t.Fatal("expected empty render for sync messages")
	}
}

func TestRenderMessageJoin(t *testing.T) {
	msg := network.Message{
		Type:      "join",
		Username:  "esteban",
		Team:      "Redbrick",
		Text:      "joined the network",
		Timestamp: "2026-06-24T14:30:00Z",
		MessageID: "abc",
	}
	rendered := tui.RenderMessage(msg)
	if rendered == "" {
		t.Fatal("expected non-empty render for join messages")
	}
}

func TestNewModelOnlyRendersMessagesMatchingNetworkID(t *testing.T) {
	network.SetSecret("test-secret")
	t.Cleanup(func() { network.SetSecret("") })

	database := setupDB(t)
	defer database.Close()
	q := db.New(database)
	ctx := context.Background()
	now := time.Now().UTC()

	office := network.Message{
		Type:      "chat",
		Username:  "user",
		Team:      "Redbrick",
		Text:      "office message",
		Timestamp: now.Format(time.RFC3339),
		MessageID: "office-msg",
		NetworkID: "office-net",
	}
	office.Sign()

	home := network.Message{
		Type:      "chat",
		Username:  "user",
		Team:      "Redbrick",
		Text:      "home message",
		Timestamp: now.Add(time.Minute).Format(time.RFC3339),
		MessageID: "home-msg",
		NetworkID: "home-net",
	}
	home.Sign()

	untagged := network.Message{
		Type:      "chat",
		Username:  "user",
		Team:      "Redbrick",
		Text:      "untagged message",
		Timestamp: now.Add(2 * time.Minute).Format(time.RFC3339),
		MessageID: "untagged-msg",
		NetworkID: "",
	}
	untagged.Sign()

	insertMessage(t, q, ctx, office)
	insertMessage(t, q, ctx, home)
	insertMessage(t, q, ctx, untagged)

	model := tui.NewModel(database, "me", "Redbrick", nil, nil, nil, ctx, func() {}, true, true, "office-net", "dev", "nerd")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated, _ = updated.Update(tui.SyncTimeoutMsg{})
	view := updated.View()

	if !strings.Contains(view, "office message") {
		t.Fatalf("expected office message to render, view: %q", view)
	}
	if strings.Contains(view, "home message") {
		t.Fatalf("expected home message to be hidden, view: %q", view)
	}
	if strings.Contains(view, "untagged message") {
		t.Fatalf("expected untagged message (empty network_id) to be hidden when scoped to office-net, view: %q", view)
	}
}

func TestNewModelOnlyRendersMessagesSignedWithCurrentSecret(t *testing.T) {
	network.SetSecret("current-secret")
	t.Cleanup(func() { network.SetSecret("") })

	database := setupDB(t)
	defer database.Close()
	q := db.New(database)
	ctx := context.Background()
	now := time.Now().UTC()

	valid := network.Message{
		Type:      "chat",
		Username:  "valid-user",
		Team:      "Redbrick",
		Text:      "this should render",
		Timestamp: now.Format(time.RFC3339),
		MessageID: "valid-message",
	}
	valid.Sign()

	network.SetSecret("wrong-secret")
	invalid := network.Message{
		Type:      "chat",
		Username:  "invalid-user",
		Team:      "Redbrick",
		Text:      "this should not render",
		Timestamp: now.Add(time.Minute).Format(time.RFC3339),
		MessageID: "invalid-message",
	}
	invalid.Sign()
	network.SetSecret("current-secret")

	insertMessage(t, q, ctx, valid)
	insertMessage(t, q, ctx, invalid)
	insertMessage(t, q, ctx, network.Message{
		Type:      "chat",
		Username:  "unsigned-user",
		Team:      "Redbrick",
		Text:      "unsigned should not render",
		Timestamp: now.Add(2 * time.Minute).Format(time.RFC3339),
		MessageID: "unsigned-message",
	})

	model := tui.NewModel(database, "me", "Redbrick", nil, nil, nil, ctx, func() {}, true, true, "", "dev", "nerd")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated, _ = updated.Update(tui.SyncTimeoutMsg{})
	view := updated.View()

	if !strings.Contains(view, valid.Text) {
		t.Fatalf("expected valid signed message to render, view: %q", view)
	}
	if strings.Contains(view, invalid.Text) {
		t.Fatalf("expected wrong-secret message to be hidden, view: %q", view)
	}
	if strings.Contains(view, "unsigned should not render") {
		t.Fatalf("expected unsigned message to be hidden, view: %q", view)
	}
}

func TestIncomingChatMentionFlashesBanner(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	model := tui.NewModel(database, "me", "Redbrick", nil, nil, nil, ctx, func() {}, false, true, "", "dev", "off")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated, _ = updated.Update(tui.SyncTimeoutMsg{})

	mention := network.Message{
		Type:      "chat",
		Username:  "alice",
		Team:      "Redbrick",
		Text:      "hey @me can you review this?",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		MessageID: "mention-msg",
	}
	updated, _ = updated.Update(tui.IncomingNetworkMsg{Message: mention})

	view := updated.View()
	if !strings.Contains(view, "mentioned you in a message") {
		t.Fatalf("expected mention banner in view, got: %q", view)
	}
	if !strings.Contains(view, "alice") {
		t.Fatalf("expected mentioning username in banner, got: %q", view)
	}
}

func TestChatWithoutMentionDoesNotFlash(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	model := tui.NewModel(database, "me", "Redbrick", nil, nil, nil, ctx, func() {}, false, true, "", "dev", "off")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated, _ = updated.Update(tui.SyncTimeoutMsg{})

	plain := network.Message{
		Type:      "chat",
		Username:  "alice",
		Team:      "Redbrick",
		Text:      "the @method needs review, thanks matthew@work",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		MessageID: "plain-msg",
	}
	updated, _ = updated.Update(tui.IncomingNetworkMsg{Message: plain})

	if strings.Contains(updated.View(), "mentioned you in a message") {
		t.Fatalf("did not expect mention banner, got: %q", updated.View())
	}
}

func TestMentionBannerClearsOnEsc(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	ctx := context.Background()

	model := tui.NewModel(database, "me", "Redbrick", nil, nil, nil, ctx, func() {}, false, true, "", "dev", "off")
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	updated, _ = updated.Update(tui.SyncTimeoutMsg{})

	mention := network.Message{
		Type:      "chat",
		Username:  "alice",
		Text:      "@me ping",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		MessageID: "mention-msg",
	}
	updated, _ = updated.Update(tui.IncomingNetworkMsg{Message: mention})

	const bannerText = "mentioned you in a message"
	if !strings.Contains(updated.View(), bannerText) {
		t.Fatal("expected banner to be visible immediately after the mention")
	}

	// Pressing esc dismisses the banner.
	updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if strings.Contains(updated.View(), bannerText) {
		t.Fatalf("expected banner to clear after esc, got: %q", updated.View())
	}
}

func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS config (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS messages (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT NOT NULL UNIQUE,
			type       TEXT NOT NULL DEFAULT 'chat',
			username   TEXT NOT NULL,
			team       TEXT NOT NULL DEFAULT '',
			text       TEXT NOT NULL,
			timestamp  TEXT NOT NULL,
			os         TEXT NOT NULL DEFAULT '',
			signature  TEXT NOT NULL DEFAULT '',
			network_id TEXT NOT NULL DEFAULT ''
		);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func insertMessage(t *testing.T, q *db.Queries, ctx context.Context, msg network.Message) {
	t.Helper()
	if err := q.InsertMessage(ctx, db.InsertMessageParams{
		MessageID: msg.MessageID,
		Type:      msg.Type,
		Username:  msg.Username,
		Team:      msg.Team,
		Text:      msg.Text,
		Timestamp: msg.Timestamp,
		Os:        msg.OS,
		Signature: msg.Signature,
		NetworkID: msg.NetworkID,
	}); err != nil {
		t.Fatal(err)
	}
}
