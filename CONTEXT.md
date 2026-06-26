# rbchat Domain Glossary

## User
A person using rbchat. Identified by a **Username** and a **Team**.

## Username
A display name chosen by the User on first launch. Persisted locally.

## Team
An affiliation or group the User belongs to, selected from a predefined list of team names on first launch. Persisted locally. Purely cosmetic — displayed as a label next to the Username in chat messages. No filtering or scoping. Team names are hardcoded in Go source.

## Message (wire format)
Unified JSON structure for all wire traffic. The `type` field discriminates:

| type | purpose |
|------|---------|
| `chat` | A user-to-user chat message. Displayed in the viewport. |
| `sync` | Sync reply. Sent on startup to request history; peers respond by broadcasting their last 50 messages. Not displayed in the viewport on the receiving end. |
| `join` | Self-announcement after setup completes. Displayed as a system message. |

Fields: `type` (discriminator), `username`, `team`, `text`, `timestamp` (ISO 8601), `message_id` (UUID v4).

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
Local SQLite table with columns: `id` (INTEGER PK), `message_id` (TEXT, UNIQUE), `username` (TEXT), `team` (TEXT), `text` (TEXT), `timestamp` (TEXT).

## Model.messages
An in-memory `[]Message` slice. This is the source of truth for what the viewport displays. On startup, seeded from DB (last 50 messages from today). During runtime, each incoming message is appended here and also saved to DB asynchronously. Capped at 10,000 messages — trims from the front when exceeded. The DB is the full archive; the slice is a rolling window.

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
5. Peers respond by broadcasting their last 50 messages (from today) as `sync`-type messages.
6. Joining peer absorbs them silently — stores in DB, does not display in viewport.
7. When sync completes (timeout or messages received) → transition to the chat view.
8. Broadcast a `join`-type message to announce presence.

## Sync
Entirely multicast-based. A joining peer broadcasts a `sync`-type message. Existing peers respond by broadcasting their last 50 messages (also `sync` type). All peers deduplicate by `message_id`. `sync`-type messages are never displayed in the viewport.

## Event bridge
The network listener goroutine calls `program.Send(IncomingMessage{...})` directly (thread-safe in Bubble Tea) to inject parsed messages into the `Update` loop. No intermediate channel.

## Error display
If a send fails, a red `[Error] Failed to send message` line is rendered briefly in the viewport (held in `Model.err`). Clears automatically on the next successful send.

## Shutdown
Ctrl+C in the Bubble Tea view → display "Shutting down..." → cancel the listener goroutine's context → close UDP connections → close DB → `tea.Quit`.

## Peer tracking
"Peers online" count is the number of unique message senders seen within the last 60 seconds. No heartbeats, no persistent connections — purely derived from recent chat activity.
