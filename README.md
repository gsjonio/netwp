# netwp

🇺🇸 English · 🇧🇷 [Português](README.pt-BR.md)

[![CI](https://github.com/gsjonio/netwp/actions/workflows/ci.yml/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/ci.yml)
[![CodeQL](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql)
[![Dependabot](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph)
[![Go version](https://img.shields.io/github/go-mod/go-version/gsjonio/netwp)](go.mod)
[![Release](https://img.shields.io/github/v/release/gsjonio/netwp)](https://github.com/gsjonio/netwp/releases/latest)
[![License: MIT](https://img.shields.io/github/license/gsjonio/netwp)](LICENSE)

**netwp** — *Internet / Rede Well Played* ("rede" is Portuguese for network).

A terminal network manager written in Go: active local-network device discovery
(ARP), live monitoring, a full dashboard, bandwidth testing, and interface
inspection. Windows-first, portable to Linux.

## Table of Contents

- [Status](#status)
- [Architecture](#architecture)
- [Install](#install)
- [Usage](#usage)
- [Notes](#notes)
- [License](#license)

## Status

- [x] Device discovery core (ARP scan, hostname, vendor by OUI, device-class guess)
- [x] Continuous monitoring (join/leave), live TUI
- [x] Bandwidth test
- [x] Interface IP inspect (read-only)
- [x] Interface IP configure (static/DHCP, Windows only)
- [x] Linux adapter (AF_PACKET raw ARP, gateway, DNS)
- [x] Persistent device aliases (nicknames, keyed by MAC)
- [x] Live dashboard (Wi-Fi, real-time bandwidth, speedtest, devices)
- [x] Per-device latency (RTT) and internet latency, native ICMP (no admin)
- [x] Wi-Fi channel recommendation from nearby AP congestion
- [x] New-device alerts (unrecognized MAC joins flagged in monitor/dashboard)
- [x] JSON export (`netwp scan --json`)
- [x] Hostname fallback via mDNS/NetBIOS when reverse DNS has nothing
- [x] Per-device port detail (`netwp ports <ip>`)

## Architecture

Hexagonal (Ports & Adapters). The `core` package is pure domain + use cases and
never imports OS/network code; adapters implement its ports and are selected at
build time via Go build tags.

```text
cmd/netwp        composition root
internal/core    domain + ports + use cases (pure)
internal/adapter arpscan · netinfo · oui (touch the OS)
internal/tui     legible table output
```

## Install

No Go toolchain? Grab a prebuilt binary from the
[Releases page](https://github.com/gsjonio/netwp/releases/latest) instead
(Windows and Linux amd64).

Requires Go 1.24+ for the options below.

### Quick install (no clone needed)

`go install` fetches the module, builds it, and drops the binary in
`$(go env GOPATH)\bin`. Put that folder on your PATH and call it as `netwp`
from any terminal (Windows resolves the `.exe` automatically):

```powershell
go install github.com/gsjonio/netwp/cmd/netwp@latest
netwp
```

Pin a specific release instead of `@latest` if you want reproducible builds,
e.g. `go install github.com/gsjonio/netwp/cmd/netwp@v0.1.0`.

### Build from source

Clone the repo if you want to read or change the code, cross-compile, or run
the test suite:

```powershell
git clone https://github.com/gsjonio/netwp.git
cd netwp
go build -o netwp.exe ./cmd/netwp
go test ./...
```

For a smaller binary, strip the symbol table and DWARF info
(about 12 MB down to 8.8 MB):

```powershell
go build -ldflags "-s -w" -o netwp.exe ./cmd/netwp
```

`go install -ldflags "-s -w" ./cmd/netwp` (run from inside the cloned repo)
does the same, straight into `$(go env GOPATH)\bin`.

The Windows scanner uses the `SendARP` API: **no admin rights and no Npcap
required**.

### Updating

Check what you have with `netwp version`, then update the same way you
installed:

- **Quick install:** re-run `go install github.com/gsjonio/netwp/cmd/netwp@latest`
  (or the specific tag you want). It overwrites the old binary.
- **Build from source:** `git pull` then rebuild (`go build`/`go install`).
- **Prebuilt binary:** download the new one from the
  [Releases page](https://github.com/gsjonio/netwp/releases/latest) and
  replace the old file. There's no self-update mechanism.

## Usage

```powershell
netwp             # no arguments: prints usage/help (same as netwp help / --help)
netwp scan        # one-shot scan, with per-device RTT
netwp scan --json # same scan, machine-readable JSON on stdout
netwp monitor     # live TUI: devices joining/leaving in real time (q to quit)
netwp dashboard   # full dashboard: wifi + live bandwidth + speedtest + devices
netwp speedtest   # download/upload throughput
netwp iface       # active interface's IP config
netwp iface static 192.168.1.50/24 192.168.1.1 8.8.8.8  # set a static address (asks to confirm)
netwp iface dhcp                                        # switch back to DHCP (asks to confirm)
netwp alias set 192.168.1.20 "Living Room TV"  # nickname a device (by IP or MAC)
netwp alias ls                                 # list nicknames
netwp alias rm 192.168.1.20                    # remove a nickname
netwp ports 192.168.1.20                       # open ports + RTT for one device
netwp version                                  # installed version
```

## Notes

See [SECURITY.md](SECURITY.md) for scanning safety and reporting a
vulnerability.

### Data & storage

- Vendor names come from the full IEEE MA-L registry, gzipped and embedded
  in the binary (`internal/adapter/oui/data`). Refresh it with the command
  in `oui.go`.
- Device aliases live in `<user-config-dir>/netwp/aliases.json`, keyed by
  MAC so a nickname survives a DHCP-assigned IP change. Plain text, safe to
  edit by hand.
- `alias set <ip>` resolves the MAC from the last scan's cache
  (`lastscan.json`), so aliasing right after a scan is instant. Pass a MAC
  instead of an IP to skip the network entirely.

### Platform support

- **Windows** is the primary, most-verified platform: ARP scan via
  `SendARP`, ICMP via `IcmpSendEcho` — neither needs admin rights. `iface
  static`/`iface dhcp` do need an elevated terminal and always ask for a
  typed "yes"; verified end-to-end on real hardware.
- **Linux** support works but is less battle-tested: the raw-ARP scanner
  (`AF_PACKET`) needs `CAP_NET_RAW` and has been run for real on a Linux
  kernel (WSL2), but only against WSL2's default NAT network, not a full
  physical LAN. `iface static`/`dhcp` isn't implemented on Linux. CI builds
  and tests natively on Ubuntu every push.
- The dashboard's Wi-Fi panel supports English and Portuguese `netsh wlan`
  output; only the Portuguese labels are verified against live output.

### How some things work

- Hostname resolution tries reverse DNS first, then falls back to mDNS and
  NetBIOS (400ms each). Neither fallback is guaranteed — a device with
  neither a Bonjour/Avahi responder nor NetBIOS support just shows no name.
- RTT is a real ICMP echo per device; a device that answers ARP but not
  ICMP (firewalled) shows online with no RTT.
- The Wi-Fi channel suggestion is a simple congestion count over visible
  APs, not an RF planner — no signal strength, DFS, or regulatory rules.
- A machine with more than one active interface (e.g. Ethernet and Wi-Fi at
  once) is recognized as "This device" on every one of them, not just the
  interface used to pick the scan's subnet.
- The speed test hits Cloudflare's anycast `speed.cloudflare.com`, which
  auto-routes to the nearest of their ~300 edges; `netwp speedtest` prints
  which one answered (e.g. "via Cloudflare edge: GRU").
- `netwp ports <ip>` re-probes one device directly (same ports used for
  classification, reported individually) instead of a full scan. No
  port-history across runs, just the current state.
- A device join is flagged "unknown" in the activity log only when its MAC
  has no alias set.

Want to contribute? See [CONTRIBUTING.md](CONTRIBUTING.md). This project
follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE).
