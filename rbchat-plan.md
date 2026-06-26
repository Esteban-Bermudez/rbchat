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
* **Distribution:** GoReleaser (Automated cross-compilation via GitHub Actions)

## Directory Structure
Following `github.com/golang-standards/project-layout`:

```text
.
├── cmd/
│   └── rbchat/
│       └── main.go           # Application entry point. Wires DB, Network, and TUI.
├── internal/
│   ├── db/                   # sqlc-generated code (DO NOT EDIT)
│   ├── network/              # net.ListenMulticastUDP and net.DialUDP wrappers
│   └── tui/                  # Bubble Tea Model, Update, and View logic
├── sql/
│   ├── schema.sql            # SQLite table definitions
│   └── query.sql             # SQL statements for sqlc to process
├── sqlc.yaml                 # sqlc configuration
├── .goreleaser.yaml          # GoReleaser CI/CD configuration
├── go.mod
└── go.sum

```

## Core Networking Concept: Multicast UDP

* **The Frequency:** All clients bind to the reserved local multicast address `224.0.0.1:9999`.
* **Broadcasting:** When a user submits a message via the Bubble Tea `Update` loop, a command fires a JSON payload to the multicast IP via `net.DialUDP`.
* **Listening:** A background goroutine in `internal/network/` continuously reads from `net.ListenMulticastUDP`. When a message arrives, it sends a custom `tea.Msg` to the Bubble Tea event loop to safely trigger a UI redraw.

## Data Structures (JSON Payloads)

```go
// Standard chat message
type ChatMessage struct {
    Type      string `json:"type"` // "chat"
    Username  string `json:"username"`
    Text      string `json:"text"`
    Timestamp string `json:"timestamp"`
}

// Emitted on startup to request missed history
type SyncRequest struct {
    Type      string `json:"type"` // "sync_request"
    Username  string `json:"username"`
    ReplyAddr string `json:"reply_addr"` 
}

```

## User Interface Design (Bubble Tea `View`)

The UI is managed by Bubble Tea's state machine, rendering distinct styled regions (via Lipgloss) to ensure typing is never interrupted by incoming network traffic.

```text
┌───────────────────────────────────────────────────────────────┐
│ 🌐 rbchat | LAN: 224.0.0.1:9999 | 4 Peers Online              │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│ [11:30 AM] Sarah: Has anyone seen the new design docs?        │
│ [11:32 AM] Esteban: Yeah, they're in the Paved shared drive.  │
│ [11:33 AM] Mike: Hola Esteban, can you review my PR?          │
│ [11:33 AM] Esteban: Claro, checking it now.                   │
│ [11:35 AM] System: [Alex joined the network]                  │
│                                                               │
│                                                               │
│                                                               │
├───────────────────────────────────────────────────────────────┤
│ > Bom dia! Just looking at the PR now... █                    │
└───────────────────────────────────────────────────────────────┘

```

## Implementation Phases

### Phase 1: Database & Tooling (`internal/db`)

* Write `sql/schema.sql` (defining the `messages` table).
* Write `sql/query.sql` (Insert message, Fetch recent messages).
* Run `sqlc generate`.
* Initialize `modernc.org/sqlite` connection in `cmd/rbchat/main.go`.

### Phase 2: Core Networking (`internal/network`)

* Build the multicast listener.
* Build the multicast broadcaster.
* Create a Go channel bridging the network listener to Bubble Tea, converting raw UDP JSON payloads into `tea.Msg` structs.

### Phase 3: The Bubble Tea UI (`internal/tui`)

* Define the core `Model` struct (holding the text input state, the viewport for the chat log, and the DB connection).
* Implement the `Init()` function to start the network listening channel.
* Implement the `Update()` function to handle keystrokes (typing, sending) and incoming network messages (`tea.Msg`).
* Implement the `View()` function to render the layout.

### Phase 4: The Sync Protocol

* Update `Init()` to broadcast a `SYNC_REQUEST` on startup.
* Update the network listener: if a `SYNC_REQUEST` is heard, query the local SQLite DB for the last 50 messages and send them via unicast to the new peer.

### Phase 5: CI/CD Distribution

* Configure `.goreleaser.yaml`.
* Set up a GitHub Action to automatically build macOS and Windows binaries on new Git tags, attaching them to a GitHub Release.

```

```
