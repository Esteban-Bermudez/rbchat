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
