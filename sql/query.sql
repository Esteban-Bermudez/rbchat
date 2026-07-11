-- name: GetConfig :one
SELECT value FROM config WHERE key = ?;

-- name: SetConfig :exec
INSERT OR REPLACE INTO config (key, value) VALUES (?, ?);

-- name: InsertMessage :exec
INSERT INTO messages (message_id, type, username, team, text, timestamp, os, signature, network_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(message_id) DO NOTHING;

-- name: GetRecentMessagesToday :many
SELECT id, message_id, type, username, team, text, timestamp, os, signature, network_id
FROM messages
WHERE date(timestamp) = date('now')
  AND signature != ''
  AND network_id = ?
ORDER BY id DESC
LIMIT ?;

-- name: GetRecentMessagesForSync :many
SELECT id, message_id, type, username, team, text, timestamp, os, signature, network_id
FROM messages
WHERE signature != ''
  AND network_id = ?
ORDER BY id DESC
LIMIT ?;
