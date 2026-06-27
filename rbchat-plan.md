```markdown
# rbchat: Redbrick LAN Terminal Chat

## Overview
`rbchat` is a zero-configuration, serverless CLI chat application designed for rapid, localized communication. It operates entirely over the Local Area Network (LAN) using UDP Multicast, eliminating the need for central signaling servers, STUN/TURN infrastructure, or internet routing.

## Architecture & Tech Stack
To ensure maximum portability and frictionless distribution across macOS and Windows, `rbchat` is built purely in Go using a CGO-free stack and follows the standard Go project layout.

* **Language:** Go
* **Networking:** Standard Library `net` (UDP Multicast)
* **UI Framework:** `charmbracelet/bubbletea` (Elm architecture for terminal UIs) & `charmbracelet/lipgloss` (for styling)
* **Database Engine:** `modernc.org/sqlite` (Pure Go SQLite driver, bypassing CGO constraints)
* **Database ORM:** `sqlc` (Generates type-safe Go code from raw SQL queries)
* **Desktop Notifications:** `github.com/gen2brain/beeep` (Cross-platform, CGO-free)
* **Distribution:** GoReleaser (Automated cross-compilation via GitHub Actions)

## Directory Structure

```text
.
├── cmd/
│   └── rbchat/
│       └── main.go           # Application entry point. Wires DB, Network, and TUI.
├── internal/
│   ├── db/                   # sqlc-generated code (DO NOT EDIT)
│   │   └── db_init.go        # Manual DB init (DLL execution, dir creation)
│   ├── network/              # net.ListenMulticastUDP and net.DialUDP wrappers
│   │   ├── listener.go
│   │   ├── broadcaster.go
│   │   └── message.go        # Unified Message struct (chat/sync/join types)
│   └── tui/                  # Bubble Tea Model, Update, and View logic
│       ├── model.go           # Model struct, NewModel constructor
│       ├── update.go          # Init, Update, handleIncoming, respondToSync
│       ├── view.go            # View, renderMessage, styles
│       ├── setup.go           # First-launch prompts (username + team)
│       └── notify.go          # beeep.Notify wrapper
├── tests/
│   ├── db/
│   ├── network/
│   └── tui/
├── sql/
│   ├── schema.sql            # SQLite table definitions
│   └── query.sql             # SQL statements for sqlc to process
├── sqlc.yaml                 # sqlc configuration
├── .goreleaser.yaml          # GoReleaser CI/CD configuration
├── .github/workflows/ci.yml  # CI + release pipeline
├── install.sh                # curl-to-sh installer
├── go.mod
└── go.sum
```

## Core Networking Concept: Multicast UDP

* **Address:** All clients bind to the reserved local multicast address `224.0.0.1:9999`.
* **Broadcasting:** When a user submits a message via the Bubble Tea `Update` loop, a command fires a JSON payload to the multicast IP via `net.DialUDP`.
* **Listening:** A background goroutine in `internal/network/` continuously reads from `net.ListenMulticastUDP`. When a message arrives, it calls `program.Send()` to inject a custom `tea.Msg` into the Bubble Tea event loop. This is thread-safe — never call UI functions from goroutines.
* **Loopback:** The sender receives their own broadcast (listener picks it up). This is harmless — dedup by `message_id` prevents duplicates.

## Data Structures (JSON Payloads)

```go
// Unified wire format — all traffic uses this struct.
// The type field discriminates the purpose.
type Message struct {
    Type      string `json:"type"`      // "chat", "sync", or "join"
    Username  string `json:"username"`
    Team      string `json:"team"`
    Text      string `json:"text"`
    Timestamp string `json:"timestamp"`  // ISO 8601
    MessageID string `json:"message_id"` // UUID v4, used for dedup
}
```

Message types:

| type | direction | purpose |
|------|-----------|---------|
| `chat` | bidirectional | A user-to-user chat message. Displayed in the viewport. |
| `sync` | inbound (history replay) | Sent on startup to request history; peers respond by broadcasting their last 50 messages. Never displayed in the viewport on the receiving end. Absorbed silently into DB. |
| `join` | broadcast | Self-announcement after setup completes. Displayed as a system message in the viewport. |

## User Interface Design (Bubble Tea `View`)

The UI is managed by Bubble Tea's state machine, rendering distinct styled regions:

```text
┌────────────────────────────────────────────────────────────────┐
│  rbchat | 224.0.0.1:9999 | 🔔 | 3 peers                      │  ← purple title bar
├────────────────────────────────────────────────────────────────┤
│                                                                │
│  [Jun 24 14:30] Esteban (Paved): Has anyone seen the new docs?  │
│  [Jun 24 14:32] Esteban (Redbrick): Yeah, in the shared drive.│
│  [Jun 24 14:35] Esteban (Duplex) joined the network              │
│                                                                │
│                                                                │
├────────────────────────────────────────────────────────────────┤
│  ctrl+n: toggle notifications                                  │  ← help text
│ > Bom dia! Just looking at it now... █                         │  ← input bar
└────────────────────────────────────────────────────────────────┘
```

Key layout details:
- Title bar includes address, notification bell indicator (green=on, red=off), and active peer count
- All styled with lipgloss; title uses purple background (#7C3AED) spanning the full line width
- The bell emoji is a separate styled segment to avoid ANSI-reset gaps
- Help text below input shows available shortcuts
- Ctrl+N toggles desktop notifications; can also be disabled at startup with `--no-notify`

## Implementation Phases

### Phase 1: Database & Tooling (`internal/db`)

* Write `sql/schema.sql` (defining the `messages` and `config` tables).
* Write `sql/query.sql` (InsertMessage, GetRecentMessagesToday, GetConfig, SetConfig).
* Run `sqlc generate`.
* Write `internal/db/db_init.go` — create DB directory, run DDL, open connection.
* Initialize `modernc.org/sqlite` connection in `cmd/rbchat/main.go`.

### Phase 2: Core Networking (`internal/network`)

* Build the multicast listener (`listener.go`).
* Build the multicast broadcaster (`broadcaster.go`).
* Define the unified `Message` struct (`message.go`).
* Wire listener to call `program.Send(IncomingMessage{...})` when a UDP packet arrives.

### Phase 3: The Bubble Tea UI (`internal/tui`)

* Define the core `Model` struct (text input, viewport for chat log, DB connection, dedup set, peer tracking map, notifications flag).
* Build `NewModel()` constructor that loads last 50 messages from DB, applies `CGO_ENABLED=0` workaround for sqlite.
* Implement `Init()` — broadcast sync request on startup.
* Implement `Update()` — handle keystrokes (typing, sending), incoming network messages, sync transition, Ctrl+C shutdown.
* Implement `View()` — title bar, viewport, help text, input bar.
* Implement first-launch setup prompts (`setup.go`) using `fmt.Print`/`bufio.Scanner` before Bubble Tea starts.

**Critical constraint:** `Init()` in Bubble Tea receives a value copy of the model. All initialization must happen in `NewModel()` (before `tea.NewProgram()`).

### Phase 4: The Sync Protocol

* On startup, broadcast a `sync`-type message to request history.
* Existing peers receive this and respond by broadcasting their last 50 messages from today (also `sync` type).
* The joining peer stores them in DB but does not display them in the viewport.
* Sync is entirely multicast-based — no unicast, no `ReplyAddr`.
* Deduplication via `message_id` + `ON CONFLICT(message_id) DO NOTHING`.
* Sync completes after a 2-second timeout or when messages arrive and the timer expires.
* After sync, broadcast a `join`-type message to announce presence.

### Phase 5: CI/CD Distribution

* Configure `.goreleaser.yaml` — cross-compile for darwin/linux/windows, flattened archives.
* GitHub Action (`.github/workflows/ci.yml`) — run `go vet`, build, and test on push/PR; create GitHub release on `v*` tag.
* Write `install.sh` — curl-to-sh installer that fetches latest release and extracts to `~/.local/bin/`.

```
```
