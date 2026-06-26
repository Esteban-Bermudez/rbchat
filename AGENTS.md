# AGENTS.md — rbchat

## Project
Zero-config LAN chat over UDP multicast. Built in Go, CGO-free, using Bubble Tea TUI + sqlc + modernc.org/sqlite.

**Read `rbchat-plan.md` and `CONTEXT.md` first** — the plan defines the architecture and phases; CONTEXT.md has the resolved domain glossary and design decisions.

## Stack specifics
- **CGO strictly forbidden.** `CGO_ENABLED=0` must be set for all builds. Use `modernc.org/sqlite` (not `mattn/go-sqlite3`).
- **sqlc**: type-safe Go from raw SQL. Edit `sql/schema.sql` and `sql/query.sql`, run `sqlc generate` to produce `internal/db/`. Never hand-edit `internal/db/`.
- **Bubble Tea**: Elm-arch TUI. Network messages enter the `Update` loop as custom `tea.Msg` values; never call UI functions from goroutines.

## Commands
```sh
go run ./cmd/rbchat                  # run the app
go build ./cmd/rbchat                # build binary
sqlc generate                        # regenerate db layer from sql/
CGO_ENABLED=0 go build ./cmd/rbchat  # verify CGO-free (CI will enforce)
gofmt -w .                           # format all Go source before every commit
```

## Tooling
- **LSP**: `gopls` must be used for Go code intelligence. OpenCode is configured to use it.
- **Formatting**: Run `gofmt -w .` on any changed files before committing. All Go code must be `gofmt`-compliant.
- **Style**: Follow idiomatic Go throughout — prefer `go vet`-clean code, use standard library where possible, avoid unnecessary abstractions and external dependencies.

## Directory structure (intended — scaffold as you go)
```
cmd/rbchat/main.go          # entrypoint, wires DB/network/TUI
internal/db/                # sqlc generated — DO NOT EDIT
internal/network/           # multicast listener + broadcaster
internal/tui/               # Bubble Tea Model/Update/View
tests/network/              # tests mirror internal/ structure (like Rails spec/)
tests/tui/
tests/db/
sql/schema.sql              # SQLite DDL
sql/query.sql               # queries for sqlc
sqlc.yaml
```

## Testing
- Follow TDD (red-green-refactor) using the `/tdd` skill.
- Tests live in `/tests/` mirroring the structure under `/internal/` (e.g. `internal/network/` → `tests/network/`).

## Implementation order (from rbchat-plan.md)
1. **DB & tooling** — schema + queries + sqlc generate + init in main
2. **Core networking** — multicast listener + broadcaster + channel bridge to tea.Msg
3. **Bubble Tea UI** — Model, Init, Update, View
4. **Sync protocol** — startup sync request + multicast history reply
5. **CI/CD** — GoReleaser + GitHub Actions

## Git conventions
- Follow https://cbea.ms/git-commit/: short subject (≤50 chars), blank line, body with motivation. No semantic prefixes ("feat:", "fix:", etc.) and no co-author trailers.
- Commit frequently, one logical change per commit. Every commit must compile and run (no WIP / broken intermediate states).
- Each commit should be easy to review, revert, or rebase independently. Avoid coupling unrelated changes in the same commit.
- No commit should contain hand-edits to `internal/db/` (it's generated).
