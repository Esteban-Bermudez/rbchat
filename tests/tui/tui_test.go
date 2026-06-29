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

	model := tui.NewModel(database, "me", "Redbrick", nil, nil, nil, ctx, func() {}, true, true)
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
			signature  TEXT NOT NULL DEFAULT ''
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
		Signature: msg.Signature,
	}); err != nil {
		t.Fatal(err)
	}
}
