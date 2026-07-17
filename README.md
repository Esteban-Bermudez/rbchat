# rbchat

Zero-configuration LAN chat over UDP multicast. No server, no sign-up, no internet — just a terminal and a local network.

<img width="2964" height="606" alt="image" src="https://github.com/user-attachments/assets/96922b7b-daf9-4ab3-bf11-7a646bbe70d3" />


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

### Keybinds

| Key | Action |
|-----|--------|
| Enter | Send message |
| Ctrl+C | Quit |
| Ctrl+N | Toggle desktop notifications on/off |

Pass `--no-notify` at startup to disable notifications entirely.

### Mentions

Type `@username` in a message to mention someone. When they receive it, a banner appears at the top of their terminal — `🔔 <you> mentioned you in a message` — for three seconds, then disappears. Matching is case-insensitive and requires a word boundary, so `@matt` won't fire for `@matthew`. Mentions work regardless of whether desktop notifications are enabled.

### Notifications on macOS

rbchat uses macOS's native notification system via `osascript`. On first run your terminal app (Terminal / iTerm2) may need notification permission:

1. Open **System Settings → Notifications**
2. Find your terminal app in the list
3. Toggle **Allow Notifications** on

If you miss the prompt, the setting is under **System Settings → Privacy & Security → Notifications**.

### Teams

Available teams: Animoto, Delivra, Duplex, Leadpages, Paved, Shift, Redbrick. Each team has a color-coded label in chat messages.

| Team | Color |
|------|-------|
| Animoto | Yellow |
| Delivra | Cyan |
| Duplex | Orange |
| Leadpages | Purple |
| Paved | Green |
| Shift | Blue |
| Redbrick | Red |

### Data

Messages are persisted locally in `~/.local/share/rbchat/rbchat.db` (respects `$XDG_DATA_HOME`). Joining peers automatically sync recent chat messages.

## Development

### Prerequisites

- Go 1.21+
- [sqlc](https://sqlc.dev) (for regenerating the DB layer)

### Commands

```sh
CGO_ENABLED=0 go run ./cmd/rbchat      # run the app
CGO_ENABLED=0 go test ./tests/...      # run all tests
sqlc generate                           # regenerate internal/db/ from sql/
gofmt -w .                              # format all Go source
```

> **Isolated testing**: Set `XDG_DATA_HOME` to run with a separate database:
> ```sh
> XDG_DATA_HOME=/tmp/rbchat-test go run ./cmd/rbchat
> ```

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
