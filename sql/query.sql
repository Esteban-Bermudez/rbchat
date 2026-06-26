-- name: GetConfig :one
SELECT value FROM config WHERE key = ?;

-- name: SetConfig :exec
INSERT OR REPLACE INTO config (key, value) VALUES (?, ?);

-- name: InsertMessage :exec
INSERT INTO messages (message_id, type, username, team, text, timestamp)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(message_id) DO NOTHING;

-- name: GetRecentMessagesToday :many
SELECT id, message_id, type, username, team, text, timestamp
FROM messages
WHERE date(timestamp) = date('now')
ORDER BY id DESC
LIMIT ?;

-- name: GetRecentMessagesForSync :many
SELECT id, message_id, type, username, team, text, timestamp
FROM messages
WHERE date(timestamp) = date('now')
ORDER BY id DESC
LIMIT ?;
