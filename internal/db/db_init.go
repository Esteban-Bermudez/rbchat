package db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Init(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "rbchat.db")
	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := database.ExecContext(context.Background(), schema); err != nil {
		return nil, err
	}
	if _, err := database.ExecContext(context.Background(), "PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}
	return database, nil
}

const schema = `
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
`
