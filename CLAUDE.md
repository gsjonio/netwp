# README Generation Standard — netwp

Instructions Claude Code must follow whenever it creates or updates a README in
this repo. Adapted from the Standard Readme Spec
(https://github.com/RichardLitt/standard-readme/blob/main/spec.md) for a **Go
terminal application** (not the original Python/FastAPI target).

## When this applies

Trigger on: "create a README", "update the README", "document this repo", or any
task that touches `README.md` / `README.pt-BR.md`.

## Reference standard

Base the structure on Standard Readme, adapted for a Go CLI. Apply KISS: include
a section only if it adds real value for this repo. Do not force empty sections.

## File layout (bilingual, root-level per Standard Readme i18n)

- `README.md` — English. Canonical, default-language file.
- `README.pt-BR.md` — Portuguese (Brazil). Structurally identical to `README.md`:
  same section order, same headings translated.
- Both link to each other at the very top:
  - EN: `🇧🇷 [Português](README.pt-BR.md)`
  - PT: `🇺🇸 [English](README.md)`
- One language per file. Never merge both into one file.
- `LICENSE` stays at the repo root (never moved into a subfolder): GitHub's
  license detection only recognizes `/LICENSE`. READMEs link it as `[MIT](LICENSE)`.
- Keep both files in sync. A content change in one must be mirrored in the other
  in the same commit.

## Section order (use only what applies)

1. **Title (H1) + language link + one-line tagline** — what the tool does, in one
   sentence. No filler like "A tool for developers".
2. **Badges** (optional) — only badges that are actually wired up. No placeholders.
3. **Table of Contents** — required only if the README exceeds ~100 lines; link
   every H2.
4. **Description / Overview** — what problem the tool solves and why it exists.
   2 to 4 paragraphs. Documentation, not marketing.
5. **Status** — checklist of shipped vs planned features. Mark honestly; if a
   feature is untested on real hardware, say so (see Style rules).
6. **Tech Stack** — bullet list (Go, stdlib `net`/`syscall`, bubbletea + lipgloss
   for the TUI). Note the hexagonal (ports/adapters) design.
7. **Prerequisites** — Go toolchain version (match `go.mod`), and platform notes:
   Windows uses `SendARP` (no admin, no Npcap); Linux raw ARP needs `CAP_NET_RAW`.
8. **Installation / Build** — copy-paste-ready commands only, no unexplained steps:
   ```powershell
   go build -o netwp.exe ./cmd/netwp        # or: go install ./cmd/netwp
   ```
   Mention `-ldflags "-s -w"` for a smaller binary.
9. **Usage** — how to run each `netwp` subcommand (scan, monitor, dashboard,
   speedtest, iface, alias), one short comment per command.
10. **Architecture** — for this CLI there is no HTTP/Swagger docs; instead describe
    the hexagonal layout: `internal/core` is pure domain + ports; adapters
    implement them and are selected by Go build tags.
11. **Project Structure** — a short tree of meaningful folders only
    (`cmd/netwp`, `internal/core`, `internal/adapter/*`, `internal/tui`), not
    every file.
12. **Testing** — `go test ./...` and `go vet ./...`. Note that `-race` needs cgo
    (no C toolchain here), so it is not part of the standard check.
13. **Notes / Security** — only concrete flags: active scanning can look intrusive
    (scan only authorized networks); `iface static`/`dhcp` need admin and confirm
    first; where data is stored (`<user-config-dir>/netwp/`).
14. **License** — link `[MIT](LICENSE)`.

## Style rules

- Plain language; assume the reader has never seen this codebase.
- Every code block must actually run. No invented commands, no unverified flags:
  validate each command against this repo before finishing.
- Prose avoids em dashes and marketing slogans (project convention from a prior
  humanizer pass). Use colons, parentheses, or shorter sentences.
- Consistent heading depth; do not jump from `#` to `####`.
- No walls of text; break up with headers and bullets.
- Verify every relative link resolves against the real repo structure.
- DRY/KISS: do not restate the same explanation in two sections. A short,
  accurate README beats a long, padded one.
- Be honest about maturity: mark cross-compiled-but-unrun paths (Linux adapters)
  and fixture-only-verified paths (Wi-Fi connected fields) as such.

## Before finishing

- [ ] `README.md` and `README.pt-BR.md` have identical section order and content.
- [ ] Every command in Installation and Usage was run/validated against this repo.
- [ ] Table of Contents links (if present) all resolve.
- [ ] No section documents something that does not exist yet in the code.
- [ ] `LICENSE` is still at the repo root and both READMEs link it as `[MIT](LICENSE)`.
