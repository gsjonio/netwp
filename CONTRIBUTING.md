# Contributing to netwp

This is a small, personal Go CLI project, but pull requests are welcome.
This project follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## Setup

Requires Go 1.24+.

```powershell
git clone https://github.com/gsjonio/netwp.git
cd netwp
go build ./...
go test ./...
go vet ./...
```

`gofmt -l .` should print nothing; run `gofmt -w <file>` on anything it
flags before committing.

## Architecture

netwp is hexagonal (ports & adapters):

- `internal/core` is pure domain: use cases, types, and the port interfaces
  they depend on. It must never import `net`, `os/exec`, `syscall`, or any
  OS-touching package directly — only stdlib types like `net.IP` as plain
  data. This is what keeps `core`'s tests fast and OS-independent.
- `internal/adapter/*` implements those ports. Platform-specific code is
  split by Go build tags (`_windows.go`, `_linux.go`, `_other.go`), selected
  at compile time, never at runtime.
- `internal/tui` renders `core` types to the terminal (plain tables and the
  bubbletea/lipgloss dashboard).
- `cmd/netwp` is the composition root: it wires concrete adapters into
  `core` use cases and parses CLI arguments. Keep it thin.

If you're adding a new capability, ask: does this belong in `core` as a new
port + use case (if it's domain logic), or in an adapter (if it talks to
the OS/network)? Keep that boundary intact.

## Before opening a PR

- [ ] `go build ./...`, `go vet ./...`, and `go test ./...` all pass.
- [ ] `gofmt -l .` is empty.
- [ ] New non-trivial logic (a branch, a parser, a heuristic) has a test.
- [ ] If you touched a documented behavior, update **both**
      [README.md](README.md) and [README.pt-BR.md](README.pt-BR.md) in the
      same PR — they must stay structurally identical (same sections, same
      content, one in English and one in Portuguese).
- [ ] Commit messages are short and imperative, prefixed by type:
      `feat: ...`, `fix: ...`, `docs: ...`, `refactor: ...`, `chore: ...`.

## Scope

Keep changes focused: one logical change per PR. If you're not sure
whether a feature fits the project (e.g. it adds a new external dependency,
or a large new subsystem), open an issue to discuss it first rather than
sending a large PR speculatively.
