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
			Signature: "testsig",
			NetworkID: "",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	messages, err := q.GetRecentMessagesForSync(ctx, db.GetRecentMessagesForSyncParams{
		NetworkID: "",
		Limit:     10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
}

func TestInsertAndGetMessagesFilteredByNetwork(t *testing.T) {
	database := setupDB(t)
	defer database.Close()
	q := db.New(database)
	ctx := context.Background()

	row := database.QueryRow("SELECT date('now')")
	var today string
	row.Scan(&today)

	for i := 0; i < 3; i++ {
		err := q.InsertMessage(ctx, db.InsertMessageParams{
			MessageID: "net1-msg" + itoa(i),
			Type:      "chat",
			Username:  "user",
			Team:      "team",
			Text:      "office " + itoa(i),
			Timestamp: today + "T10:0" + itoa(i) + ":00Z",
			Signature: "testsig",
			NetworkID: "office-net",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 2; i++ {
		err := q.InsertMessage(ctx, db.InsertMessageParams{
			MessageID: "net2-msg" + itoa(i),
			Type:      "chat",
			Username:  "user",
			Team:      "team",
			Text:      "home " + itoa(i),
			Timestamp: today + "T11:0" + itoa(i) + ":00Z",
			Signature: "testsig",
			NetworkID: "home-net",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	msgs, err := q.GetRecentMessagesForSync(ctx, db.GetRecentMessagesForSyncParams{
		NetworkID: "office-net",
		Limit:     100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 office-net messages, got %d", len(msgs))
	}
	for _, m := range msgs {
		if m.NetworkID != "office-net" {
			t.Fatalf("expected network_id office-net, got %s", m.NetworkID)
		}
	}

	homeMsgs, err := q.GetRecentMessagesForSync(ctx, db.GetRecentMessagesForSyncParams{
		NetworkID: "home-net",
		Limit:     100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(homeMsgs) != 2 {
		t.Fatalf("expected 2 home-net messages, got %d", len(homeMsgs))
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

	insert := db.InsertMessageParams{
		MessageID: "dup1",
		Type:      "chat",
		Username:  "user",
		Team:      "team",
		Timestamp: today + "T10:00:00Z",
		Signature: "testsig",
		NetworkID: "",
	}
	insert.Text = "first"
	err := q.InsertMessage(ctx, insert)
	if err != nil {
		t.Fatal(err)
	}

	insert.Text = "duplicate"
	err = q.InsertMessage(ctx, insert)
	if err != nil {
		t.Fatal(err)
	}

	messages, err := q.GetRecentMessagesForSync(ctx, db.GetRecentMessagesForSyncParams{
		NetworkID: "",
		Limit:     10,
	})
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
