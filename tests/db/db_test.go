package db_test

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/esteban/rbchat/internal/db"
)

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
			timestamp  TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func TestSetAndGetConfig(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	q := db.New(database)
	ctx := context.Background()

	err := q.SetConfig(ctx, db.SetConfigParams{Key: "username", Value: "testuser"})
	if err != nil {
		t.Fatal(err)
	}

	val, err := q.GetConfig(ctx, "username")
	if err != nil {
		t.Fatal(err)
	}
	if val != "testuser" {
		t.Fatalf("expected testuser, got %s", val)
	}
}

func TestGetConfigMissingKey(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	q := db.New(database)
	ctx := context.Background()

	_, err := q.GetConfig(ctx, "nonexistent")
	if err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}
}

func TestInsertAndGetRecentMessages(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	q := db.New(database)
	ctx := context.Background()

	// get today's date in SQLite format
	row := database.QueryRow("SELECT date('now')")
	var today string
	row.Scan(&today)

	for i := 0; i < 3; i++ {
		err := q.InsertMessage(ctx, db.InsertMessageParams{
			MessageID: "msg" + itoa(i),
			Type:      "chat",
			Username:  "user",
			Team:      "team",
			Text:      "hello " + itoa(i),
			Timestamp: today + "T1" + itoa(i) + ":00:00Z",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	messages, err := q.GetRecentMessagesForSync(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
}

func TestInsertDuplicateMessage(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	q := db.New(database)
	ctx := context.Background()

	row := database.QueryRow("SELECT date('now')")
	var today string
	row.Scan(&today)

	err := q.InsertMessage(ctx, db.InsertMessageParams{
		MessageID: "dup1",
		Type:      "chat",
		Username:  "user",
		Team:      "team",
		Text:      "first",
		Timestamp: today + "T10:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	err = q.InsertMessage(ctx, db.InsertMessageParams{
		MessageID: "dup1",
		Type:      "chat",
		Username:  "user",
		Team:      "team",
		Text:      "duplicate",
		Timestamp: today + "T10:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}

	messages, err := q.GetRecentMessagesForSync(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message after dedup, got %d", len(messages))
	}
}

func itoa(i int) string {
	return string(rune('0' + i))
}
