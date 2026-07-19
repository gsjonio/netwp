# netwp

🇺🇸 English · 🇧🇷 [Português](README.pt-BR.md)

[![CI](https://github.com/gsjonio/netwp/actions/workflows/ci.yml/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/ci.yml)
[![CodeQL](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/github-code-scanning/codeql)
[![Dependabot](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/dependabot/update-graph)
[![Go version](https://img.shields.io/github/go-mod/go-version/gsjonio/netwp)](go.mod)
[![Release](https://img.shields.io/github/v/release/gsjonio/netwp)](https://github.com/gsjonio/netwp/releases/latest)
[![License: MIT](https://img.shields.io/github/license/gsjonio/netwp)](LICENSE)
[![Wiki](https://img.shields.io/badge/docs-wiki-blue?logo=github)](https://github.com/gsjonio/netwp/wiki)
[![Buy Me a Coffee](https://img.shields.io/badge/Buy_Me_a_Coffee-gugamenezes-FFDD00?logo=buymeacoffee&logoColor=black)](https://buymeacoffee.com/gugamenezes)

**netwp**: *Internet / Rede Well Played* ("rede" is Portuguese for network).

A terminal network manager written in Go: active local-network device discovery
(ARP), live monitoring, a full dashboard, bandwidth testing, and interface
inspection. Windows-first, portable to Linux.

New to networking? Start with the [beginner's guide](docs/GUIDE.md)
([pt-BR](docs/GUIDE.pt-BR.md)) instead: it explains every term and table
column in plain language. The [wiki](https://github.com/gsjonio/netwp/wiki) has
a full command reference, FAQ, and troubleshooting.

## Table of Contents

- [Features](#features)
- [Install](#install)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Usage](#usage)
- [Notes](#notes)
- [Support](#support)
- [License](#license)

## Features

**Discovery & monitoring.** Active ARP scan with hostname (reverse DNS, then
mDNS/NetBIOS fallback), vendor by OUI, device-class guess, per-device RTT and
TTL (with an OS-family hint) and open-port detail (sensitive ones like
SSH/SMB/RDP flagged), all continuously tracked in a live TUI with new-device
alerts.

**Dashboard.** Wi-Fi, real-time bandwidth, speedtest and devices in one live
view, with Wi-Fi channel recommendations from nearby AP congestion.

**Interface & network config.** Read-only IP inspection everywhere; static/DHCP
configuration on Windows. Linux support via raw ARP (`AF_PACKET`).

**Persistence & tooling.** Device aliases that survive DHCP IP changes, JSON
export (`netwp scan --json`), and self-update (`netwp update` / `netwp
version`).

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

It's a thin wrapper around `go install github.com/gsjonio/netwp/cmd/netwp@latest`:
the same command as below, without retyping the module path. Overwriting the
running binary works even on Windows.

Otherwise, update the same way you installed:

- **Quick install:** re-run `go install github.com/gsjonio/netwp/cmd/netwp@latest`
  (or the specific tag you want). It overwrites the old binary.
- **Build from source:** `git pull` then rebuild (`go build`/`go install`).
- **Prebuilt binary:** download the new one from the
  [Releases page](https://github.com/gsjonio/netwp/releases/latest) and
  replace the old file. There's no self-update mechanism for this path.

## Architecture

netwp is hexagonal (Ports & Adapters). `internal/core` is the pure domain: the
use cases (device discovery, classification, scan diffing, the connectivity
doctor) and the small port interface each one depends on. It never imports
`net`, `os/exec`, or `syscall`, only plain data types, so the whole domain runs
against fakes in tests without touching a real network card.

Adapters in `internal/adapter/*` implement those ports against the real system.
The platform-specific ones (ARP scan, ICMP ping, interface config, Wi-Fi info)
are chosen at build time by Go build tags, never at runtime: on Windows the ARP
scan is `SendARP` and ping is `IcmpSendEcho`; on Linux the scanner is a raw
`AF_PACKET` socket. The core cannot tell which one it is talking to.

`cmd/netwp` is the composition root: it wires concrete adapters into the core
use cases and dispatches the CLI. `internal/tui` renders core types to the
terminal (the scan table, the live monitor, and the dashboard).

A scan flows one way: `cmd` builds a `core.Discovery` from the adapters and
calls `Run`; the use case enriches each host (hostname, vendor, open ports, RTT,
mDNS services) concurrently, classifies it, and hands the result to
`internal/tui`.

## Project Structure

The layout follows the standard Go `cmd` + `internal` split:

```text
cmd/netwp         composition root: CLI dispatch + adapter wiring
internal/core     pure domain: use cases + port interfaces (no OS/net imports)
internal/adapter  adapters that touch the OS/network (arpscan, icmpping,
                  netinfo, oui, tcpprobe, namelookup, wifi, ...)
internal/tui      terminal rendering: scan table, monitor, dashboard
```

## Usage

| Command | What it does |
| --- | --- |
| *(none)* / `help` / `-h` / `--help` | Print usage |
| `scan` / `scan --json` / `scan --diff` / `scan --ports=<list>` / `scan --class=<class>` | One-shot scan, with per-device RTT; `--json` for machine-readable output, `--diff` to print only what changed since the last scan, `--ports=22,80,443` to probe a custom TCP port set, `--class=media` to show only devices of one class (router/computer/mobile/media/printer/iot) |
| `monitor` / `monitor --alert-down=<rate>` / `monitor --quiet` | Live TUI: devices joining/leaving in real time (`q` to quit); `--alert-down` flags a download rate drop, e.g. `--alert-down=50Mbps`; `--quiet` runs headless (no UI), one line per event to stdout for a service or logfile |
| `dashboard` | Full dashboard: Wi-Fi + live bandwidth + speedtest + devices + an operations log |
| `speedtest` / `speedtest --json` | Download/upload throughput; `--json` for machine-readable output |
| `iface` | Inspect the active interface's IP config |
| `iface static <ip>/<bits> <gw> [dns...]` | Set a static address (asks to confirm) |
| `iface dhcp` | Switch back to DHCP (asks to confirm) |
| `alias set <ip\|mac> <name>` / `ls [--json]` / `rm <ip\|mac>` | Nickname a device / list / remove |
| `class set <ip\|mac> <class>` / `ls [--json]` / `rm <ip\|mac>` | Pin a device's class when the guess is wrong (router/computer/mobile/media/printer/iot) |
| `watch add <ip\|mac>` / `ls [--json]` / `rm <ip\|mac>` | Alert (highlight + bell) when a device leaves during monitor/dashboard |
| `ports <ip>` / `ports <ip> --json` | Open ports + RTT + TTL for one device; `--json` for machine-readable output |
| `wake <ip\|mac\|alias>` | Send a Wake-on-LAN magic packet to power on a device |
| `doctor` / `doctor --json` | Diagnose connectivity: interface, gateway, internet, DNS, Wi-Fi; `--json` for machine-readable output |
| `events [n]` / `events --device=<x>` / `events --json` | Print the last n join/leave events (default 20); `--device=<alias-or-mac>` filters to one device; `--json` for machine-readable output |
| `version` | Installed version |
| `update` | Update to the latest version (needs Go) |
| `uninstall` | Remove netwp's local data (asks to confirm); prints how to remove the binary |

```powershell
netwp scan --json | ConvertFrom-Json | Where-Object reachable
netwp alias set 192.168.1.20 "Living Room TV"
```

The CLI is built on [cobra](https://github.com/spf13/cobra): every command has
its own `--help` (e.g. `netwp scan --help`), and `netwp completion <bash|zsh|fish|powershell>`
generates a shell-completion script.

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
  `SendARP`, ICMP via `IcmpSendEcho`, neither needing admin rights. `iface
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

New to terms like MAC, TTL, or "unknown device"? The
[beginner's guide](docs/GUIDE.md) ([pt-BR](docs/GUIDE.pt-BR.md)) explains
what everything on screen means. The notes below are implementation trivia for
people who already know networking.

#### Discovery & classification

- Hostname resolution falls back to mDNS/NetBIOS when reverse DNS has
  nothing; some devices still won't show a name. Mechanics are in
  [CONTRIBUTING.md](CONTRIBUTING.md).
- RTT and TTL come from the same ICMP echo per device, so a firewalled
  device (answers ARP but not ICMP) shows online with neither.
- A machine with more than one active interface (e.g. Ethernet and Wi-Fi at
  once) is recognized as "This device" on all of them.
- The CLASS guess combines advertised mDNS services (a Chromecast, printer,
  or iPhone announces what it is), then ~29 probed ports, then vendor. When
  it's still wrong (a phone with a random MAC and no open ports), pin it with
  `netwp class set <ip|mac> <class>`; a manual pin always wins.

#### Monitor & dashboard

- Press `/` to filter the device table by a substring of any field (IP,
  alias, hostname, vendor, MAC, class); Enter keeps the filter, Esc clears
  it. The online/known counts still reflect the whole network.
- Press `s` to cycle the sort column (IP, RTT, name, class); online devices
  always sort ahead of offline ones, so `s` orders within each group. The
  active column shows in the footer.
- Two events ring the terminal bell and highlight their log line: an
  unrecognized device joining (no alias set), and a `netwp watch`-listed
  device leaving. Everything else stays quiet.
- The DEVICES panel shows a per-class breakdown of what's online (e.g.
  "2 Media · 1 Router"), skipping "This device" and unclassified hosts.
- The LOG panel (bottom) traces the dashboard's own work: scans starting and
  finishing, speedtests, and internet/Wi-Fi state changes. On a short terminal
  it shrinks, then hides, so the device table and footer keep priority.
  (Distinct from the ACTIVITY panel, which lists device joins/leaves.)
- The Wi-Fi channel suggestion is a simple congestion count over visible APs,
  not an RF planner.
- `netwp monitor --alert-down=<rate>` (e.g. `50Mbps`) highlights the
  bandwidth line when download drops below that threshold. Omit it and monitor
  behaves exactly as before.
- `monitor`/`dashboard` (and `monitor --quiet`) log every join/leave to
  `<user-config-dir>/netwp/events.jsonl`; `netwp events [n]` reads them back.
  The file is bounded: once it passes ~1 MB it is trimmed to the most recent
  5000 events, so a long-running monitor can't grow it without limit.

#### Commands

- `netwp scan --diff` compares against the previous scan (identity by MAC)
  and prints only what changed, including possible IP/MAC conflicts.
- `netwp ports <ip>` re-probes one device directly instead of a full scan,
  with no port history across runs.
- `netwp wake` only powers on a device that was left with Wake-on-LAN enabled
  (a BIOS/OS setting). It broadcasts and gets no reply, so it reports "sent",
  not "woke". An alias or a cached IP resolves even while the target is off.
- `netwp doctor` checks top-down (interface → gateway → internet → DNS); the
  topmost ✗ is usually the root cause and explains the ones below it.
- The speed test hits Cloudflare's anycast `speed.cloudflare.com`; `netwp
  speedtest` prints which edge answered.
- Setting the `NO_COLOR` environment variable (to any value) turns off colored
  output everywhere — the scan table and the live views — per the
  [no-color.org](https://no-color.org) convention.

Want to contribute? See [CONTRIBUTING.md](CONTRIBUTING.md). This project
follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## Support

netwp is free and open source. If it saves you time, you can support its
development with a coffee. Thank you! ☕

[![Buy Me a Coffee](https://img.shields.io/badge/Buy_Me_a_Coffee-gugamenezes-FFDD00?style=for-the-badge&logo=buymeacoffee&logoColor=black)](https://buymeacoffee.com/gugamenezes)

## License

[MIT](LICENSE).
