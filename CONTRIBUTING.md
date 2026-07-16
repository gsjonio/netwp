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

## Branching (gitflow)

The repo follows [gitflow](https://nvie.com/posts/a-successful-git-branching-model/).
Two long-lived branches, both protected (pull requests only, no direct pushes):

- **`develop`** is the default branch and the integration target. Day-to-day
  work branches off it and merges back into it.
- **`main`** holds released code only. It receives `release/*` and `hotfix/*`
  merges, and every commit on it is tagged.

Short-lived branches:

- **`feature/<name>`** — branch off `develop`, PR back into `develop`.
- **`release/<version>`** — branch off `develop` to stabilize a release, then
  PR into `main`; tag `main` (`vX.Y.Z`), and merge back into `develop`.
- **`hotfix/<version>`** — branch off `main` for an urgent fix, PR into `main`,
  tag, then merge back into `develop`.

Pushing a `vX.Y.Z` tag triggers the release workflow, which builds the Windows
and Linux binaries and attaches them to the GitHub Release. So a normal release
is: merge the `release/*` PR into `main`, then push the tag.

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

## Implementation notes

Mechanics worth knowing before touching these paths (the README covers what
they mean for a user; this is how they work):

- **Hostname fallback** (`internal/adapter/namelookup`): reverse DNS runs
  first; on a miss, an mDNS query and a NetBIOS query are raced against each
  other, each bounded to 400ms, first non-empty answer wins. Neither is
  guaranteed — a device with no Bonjour/Avahi responder and no NetBIOS
  support just returns nothing.
- **Classification probe** (`internal/adapter/tcpprobe`): the small,
  deliberately-fixed set of well-known ports it checks is also what feeds
  `core.Classify` and what the device table's PORTS column displays. There
  is exactly one probe per device per scan; the column costs nothing extra
  because it reuses that result instead of triggering a second sweep.

## Before opening a PR

- [ ] `go build ./...`, `go vet ./...`, and `go test ./...` all pass.
- [ ] `gofmt -l .` is empty.
- [ ] New non-trivial logic (a branch, a parser, a heuristic) has a test.
- [ ] If you touched a documented behavior, update **both**
      [README.md](README.md) and [README.pt-BR.md](README.pt-BR.md) in the
      same PR — they must stay structurally identical (same sections, same
      content, one in English and one in Portuguese). If it's something a
      beginner would need explained (a new term, table column, or warning
      sign), update [docs/GUIDE.md](docs/GUIDE.md) and
      [docs/GUIDE.pt-BR.md](docs/GUIDE.pt-BR.md) too.
- [ ] Commit messages are short and imperative, prefixed by type:
      `feat: ...`, `fix: ...`, `docs: ...`, `refactor: ...`, `chore: ...`.

## Scope

Keep changes focused: one logical change per PR. If you're not sure
whether a feature fits the project (e.g. it adds a new external dependency,
or a large new subsystem), open an issue to discuss it first rather than
sending a large PR speculatively.
