# rbchat Domain Glossary

## User
A person using rbchat. Identified by a **Username** and a **Team**. Persisted locally.

## Desktop notifications
Sent via `beeep.Notify` for incoming chat messages from other users. Toggle at runtime with Ctrl+N (updates `Model.notificationsEnabled`). Disable entirely at startup with `--no-notify` flag (skips `Model.notificationsEnabled` initialization). Indicator shown in the title bar: 🔔 (green) when enabled, 🔕 (red) when disabled.

## Title bar
Three concatenated lipgloss-styled segments, each independently setting the purple background (#7C3AED), to prevent ANSI-reset gaps from the bell emoji breaking the background color for the peer count.

Segments: `left = "rbchat | {addr} | "`, `middle = bellStyle.Render("🔔"/"🔕")`, `right = " | {n} peers"`. Background color is dark slate `#1A1B26`.

## Username
A display name chosen by the User on first launch. Persisted locally.

## Team
An affiliation or group the User belongs to, selected from a predefined list of team names on first launch. Persisted locally. Displayed as a colored label next to the Username in chat messages. Each team has a distinct color:

| Team | Color |
|------|-------|
| Animoto | Yellow `#FFD700` |
| Delivra | Cyan `#00CED1` |
| Duplex | Orange `#FFA500` |
| Leadpages | Purple `#7C3AED` |
| Paved | Green `#10B981` |
| Shift | Blue `#3B82F6` |
| Redbrick | Red `#EF4444` |

Usernames are rendered in the default terminal color (no foreground override). Team names are rendered in the terminal default color with the team color applied as foreground. No filtering or scoping by team — purely cosmetic.

## Message (wire format)
Unified JSON structure for all wire traffic. The `type` field discriminates:

| type | purpose |
|------|---------|
| `chat` | A user-to-user chat message. Displayed in the viewport. |
| `sync` | Sync request. Sent on startup to request history; peers respond by broadcasting their recent chat messages. Never displayed in the viewport and never appended to m.messages — stored in DB only. |
| `join` | Self-announcement after setup completes. Displayed as a system message. |

Fields: `type` (discriminator), `username`, `team`, `text`, `timestamp` (ISO 8601), `message_id` (UUID v4), `replay` (bool, omitempty), `signature` (string, omitempty — HMAC-SHA256 hex digest).

## Message signing
Every outgoing message is signed with HMAC-SHA256 using a shared secret (`RBCHAT_SECRET`). The HMAC is computed over `message_id + type + username + team + text + timestamp` (in that order, excluding `replay` and `signature`). The resulting hex digest is set as the `signature` field.

On receipt, the listener recomputes the HMAC and verifies it against the stored signature. Messages with invalid or missing signatures are silently dropped when a secret is configured. Messages loaded from the local DB are also revalidated against the current secret before display or sync replay; a non-empty signature alone is not trusted.

The secret is injected at build time via `-ldflags -X main.rbchatSecret=...` (sourced from `RBCHAT_SECRET` env var in GoReleaser/CI). When no secret is set, signing and verification are disabled entirely — all messages pass through unchanged.

## Replay flag
A transient boolean on the wire format (`Message.Replay`). Set to `true` by `respondToSync()` for history replays. On receipt, suppresses desktop notifications and peer tracking so replaying old messages doesn't trigger alerts or inflate the online count. Never stored in the DB.

## Day dividers
The `refreshViewport()` function inserts a date divider line between messages from different days, rendered in gray (`#6B7280`). Format: `── Jan 2, 2006 ──`. The first day group has no preceding divider. Messages that render as empty (`sync` type) are skipped entirely and don't affect the date tracking.

## Timestamp
ISO 8601 on the wire and in the DB. Displayed in the viewport as `[Mon DD HH:MM]` (e.g. `[Jun 24 14:30]`). Parsed on receipt, formatted for display in `View()`.

## Setup flow (first launch)
Before Bubble Tea starts, use simple `fmt.Print`/`bufio.Scanner` prompts:
1. Prompt for username
2. Show team selection list (numbered), prompt for choice
3. Persist both to config table
4. Proceed to Bubble Tea

## TUI components
- `bubbles/textinput` for the message input bar
- `bubbles/viewport` for the scrollable chat log pane
- `lipgloss` for styling (colors, borders, layout)

## config table
Local SQLite table with columns: `key` (TEXT PK), `value` (TEXT). Stores `username`, `team`, and any future per-user settings.

## messages table
Local SQLite table with columns: `id` (INTEGER PK), `message_id` (TEXT, UNIQUE), `type` (TEXT), `username` (TEXT), `team` (TEXT), `text` (TEXT), `timestamp` (TEXT), `signature` (TEXT, DEFAULT '').

## Model.messages
An in-memory `[]Message` slice. This is the source of truth for what the viewport displays. On startup, seeded from DB (last 50 **chat** messages from today — join/sync filtered out). During runtime, all message types (chat, join) are appended as they arrive in real-time. Capped at 10,000 messages — trims from the front when exceeded. The DB is the full archive; the slice is a rolling window. Sync-type messages are never appended to m.messages (stored in DB only).

## SQL queries (sqlc)
1. `GetConfig(key)` — select value from config
2. `SetConfig(key, value)` — insert or replace into config
3. `InsertMessage(...)` — insert into messages with `ON CONFLICT(message_id) DO NOTHING` for dedup
4. `GetRecentMessagesToday(n)` — select * from messages where date(timestamp) = date('now') order by id desc limit n

## db path
`$XDG_DATA_HOME/rbchat/rbchat.db` — defaults to `~/.local/share/rbchat/rbchat.db`. Directory created on first launch.

## Startup flow
1. Open/create DB. Run schema migrations.
2. Check config for `username`. If missing → run setup (prompt for username, pick team).
3. Start Bubble Tea with a "Syncing with the mesh..." loading view.
4. Broadcast a `sync`-type message.
5. Peers respond by broadcasting their last 50 **chat** messages (from today) — join and sync messages are excluded.
6. Joining peer absorbs them silently — stores in DB, appends to m.messages. No notifications or peer tracking for replays.
7. When sync completes (2s timeout): sort m.messages chronologically by ISO 8601 timestamp string, then transition to the chat view. Viewport shows day dividers between different dates.
8. Broadcast a `join`-type message to announce presence.

## Sync
Entirely multicast-based. A joining peer broadcasts a `sync`-type message. Existing peers respond by broadcasting their last 50 **chat** messages (join/sync filtered out), preserving the original message_id for dedup but marking `Replay: true`. Sync-type messages on the wire are only ever the sync request itself — never appended to m.messages, stored in DB only.

## Event bridge
The network listener goroutine calls `program.Send(IncomingMessage{...})` directly (thread-safe in Bubble Tea) to inject parsed messages into the `Update` loop. No intermediate channel.

## Error display
If a send fails, a red `[Error] Failed to send message` line is rendered briefly in the viewport (held in `Model.err`). Clears automatically on the next successful send.

## Shutdown
Ctrl+C in the Bubble Tea view → display "Shutting down..." → cancel the listener goroutine's context → close UDP connections → close DB → `tea.Quit`.

## Peer tracking
"Peers online" count is the number of unique message senders seen within the last 60 seconds. Only `chat` and `join` messages update the timer — `sync` messages (history replays) are excluded to prevent stale peers from appearing online.
