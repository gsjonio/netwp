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
- [Install](#install)
- [Architecture](#architecture)
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
- [x] Open ports in the device table, with sensitive ones (SSH/SMB/RDP) flagged
- [x] Self-update (`netwp update`) and version reporting (`netwp version`)

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

### Privileges by command

| Command | Windows | Linux |
| --- | --- | --- |
| `scan` · `monitor` · `dashboard` | no privilege needed | needs `CAP_NET_RAW` |
| `ports` · `speedtest` · `alias` · `version` · `update` | no privilege needed | no privilege needed |
| `iface` (inspect only) | no privilege needed | no privilege needed |
| `iface static` / `iface dhcp` | needs an elevated terminal | not implemented |

Windows uses the `SendARP`/`IcmpSendEcho` APIs for `scan`, so the read-only
commands never need admin. On Linux, grant the raw-ARP scanner capability
once instead of running as root every time:

```bash
sudo setcap cap_net_raw+ep $(which netwp)
```

### Updating

Check what you have with `netwp version`. If you have the Go toolchain
(whichever way you installed netwp), the easiest path is:

```powershell
netwp update
```

It's a thin wrapper around `go install github.com/gsjonio/netwp/cmd/netwp@latest`
— same command as below, just without retyping the module path. Overwriting
the running binary works even on Windows.

Otherwise, update the same way you installed:

- **Quick install:** re-run `go install github.com/gsjonio/netwp/cmd/netwp@latest`
  (or the specific tag you want). It overwrites the old binary.
- **Build from source:** `git pull` then rebuild (`go build`/`go install`).
- **Prebuilt binary:** download the new one from the
  [Releases page](https://github.com/gsjonio/netwp/releases/latest) and
  replace the old file. There's no self-update mechanism for this path.

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

## Usage

| Command | What it does |
| --- | --- |
| *(none)* / `help` / `-h` / `--help` | Print usage |
| `scan` / `scan --json` | One-shot scan, with per-device RTT; `--json` for machine-readable output |
| `monitor` | Live TUI: devices joining/leaving in real time (`q` to quit) |
| `dashboard` | Full dashboard: Wi-Fi + live bandwidth + speedtest + devices |
| `speedtest` | Download/upload throughput |
| `iface` | Inspect the active interface's IP config |
| `iface static <ip>/<bits> <gw> [dns...]` | Set a static address (asks to confirm) |
| `iface dhcp` | Switch back to DHCP (asks to confirm) |
| `alias set <ip\|mac> <name>` / `ls` / `rm <ip\|mac>` | Nickname a device / list / remove |
| `ports <ip>` | Open ports + RTT for one device |
| `version` | Installed version |
| `update` | Update to the latest version (needs Go) |

```powershell
netwp scan --json | ConvertFrom-Json | Where-Object reachable
netwp alias set 192.168.1.20 "Living Room TV"
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
- The device table's PORTS column reuses the ports the classification probe
  already collects, so it costs no extra scanning. SSH (22), SMB (445) and
  RDP (3389) render in red: exposed on a home network they are usually
  unintentional. Port names are one level down, via `netwp ports <ip>`.
- The dashboard's DEVICES panel shows a per-class breakdown of what's online
  (e.g. "2 Media · 1 Router"). "This device" and unclassified hosts are left
  out, since neither says anything about the network.
- A device join is flagged "unknown" in the activity log only when its MAC
  has no alias set.

Want to contribute? See [CONTRIBUTING.md](CONTRIBUTING.md). This project
follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE).
