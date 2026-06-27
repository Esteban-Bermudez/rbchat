# rbchat

Zero-configuration LAN chat over UDP multicast. No server, no sign-up, no internet — just a terminal and a local network.

## Install

### One-liner (macOS, Linux)

```sh
curl -fL https://raw.githubusercontent.com/Esteban-Bermudez/rbchat/main/install.sh | sh
```

Installs to `~/.local/bin/rbchat`. Make sure `~/.local/bin` is on your `PATH` (add `export PATH="$HOME/.local/bin:$PATH"` to your shell profile if not).

> **macOS Gatekeeper**: The binary isn't signed with an Apple Developer certificate. On first run, macOS may show "rbchat cannot be opened because it is not from an identified developer." To bypass: right-click the file in Finder → Open, or run `xattr -rd com.apple.quarantine ~/.local/bin/rbchat`.

### From source

```sh
go install github.com/esteban/rbchat/cmd/rbchat@latest
```

### From a release

Download the binary for your platform from the [releases page](https://github.com/esteban/rbchat/releases), make it executable, and run it.

## Usage

```sh
rbchat
```

On first launch you'll be prompted for a username and team. After that, you're in the chat — any other `rbchat` instance on the LAN will automatically discover you.

- Type a message and press Enter to send
- Ctrl+C to quit
- Messages are persisted locally in `~/.local/share/rbchat/rbchat.db`
- Joining peers automatically sync the last 50 messages from today

### Teams

Available teams: Animoto, Delivra, Duplex, Leadpages, Paved, Shift, Redbrick. Team is purely cosmetic — displayed next to your username in chat messages.

## Development

### Prerequisites

- Go 1.21+
- [sqlc](https://sqlc.dev) (for regenerating the DB layer)

### Commands

```sh
CGO_ENABLED=0 go run ./cmd/rbchat    # run the app
CGO_ENABLED=0 go test ./tests/...    # run all tests
sqlc generate                         # regenerate internal/db/ from sql/
gofmt -w .                            # format all Go source
```

### Project structure

```
cmd/rbchat/main.go     # entrypoint
internal/
  db/                  # sqlc-generated DB layer (DO NOT EDIT)
  network/             # multicast listener + broadcaster
  tui/                 # Bubble Tea TUI
sql/
  schema.sql           # SQLite DDL
  query.sql            # sqlc queries
tests/
  db/                  # DB tests
  network/             # network tests
  tui/                 # TUI tests
```

CGO is strictly forbidden. All builds use `CGO_ENABLED=0` with `modernc.org/sqlite`.
