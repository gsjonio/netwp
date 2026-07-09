# netwp

🇧🇷 [Português](README.pt-BR.md)

[![CI](https://github.com/gsjonio/netwp/actions/workflows/ci.yml/badge.svg)](https://github.com/gsjonio/netwp/actions/workflows/ci.yml)

**netwp** stands for *Internet / Rede Well Played* ("rede" is Portuguese for network).

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

## Usage

```powershell
netwp             # one-shot scan (default), with per-device RTT
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
```

## Notes

- Vendor names come from the full IEEE MA-L registry, gzipped and embedded in
  the binary (`internal/adapter/oui/data`). Refresh it with the command in
  `oui.go`.
- Active scanning may be flagged as intrusive on managed/corporate networks.
  Only scan networks you own or are authorized to.
- Device aliases are stored as JSON in `<user-config-dir>/netwp/aliases.json`,
  keyed by MAC so a nickname sticks even when DHCP hands the device a new IP.
  The file is plain text and safe to edit by hand.
- `alias set <ip>` resolves the MAC from the last scan's cache
  (`lastscan.json`) and only re-scans on a miss, so aliasing right after a
  scan is instant. Pass a MAC instead of an IP to skip the network entirely.
- The bandwidth test uses Cloudflare's public `speed.cloudflare.com`
  endpoint: no API key, no self-hosted server. Unlike Speedtest.net's server
  list, there's no explicit "pick the nearest server" step: the endpoint is
  anycast, so the same URL always routes to whichever of Cloudflare's ~300
  edges is closest to you. `netwp speedtest` prints which one answered (e.g.
  "via Cloudflare edge: GRU") so that's verifiable, not just asserted.
- `iface static`/`iface dhcp` shell out to `netsh` and need an elevated
  (admin) terminal on Windows. They always ask for a typed "yes" before
  touching the real configuration; there's no `--yes` flag to skip it.
  Verified on real hardware in an elevated session, including an interface
  name with a space ("Ethernet 2") — a risk flagged and now confirmed fine.
  Not implemented on Linux yet.
- The dashboard's Wi-Fi panel reads `netsh wlan` (English and Portuguese
  labels supported). Verified on real hardware in both states: disconnected,
  and connected (SSID/BSSID/channel/signal/Rx-Tx rate of your own
  association). The English labels are still fixture-only, since testing
  them needs an English-locale Windows install. On a wired-only host the
  panel shows "disconnected".
- The Linux scanner (raw ARP over `AF_PACKET`) needs `CAP_NET_RAW` (root, or
  `setcap cap_net_raw+ep` on the binary). CI builds, vets, and runs the test
  suite natively on Ubuntu on every push. Beyond that, it has now sent and
  received a real ARP request/reply on a real Linux kernel (WSL2 Ubuntu),
  correctly discovering and classifying the gateway. That run was on WSL2's
  default NAT network, a different broadcast domain than the physical LAN,
  so full home-network device visibility on Linux (or WSL2 in mirrored
  networking mode) is still unconfirmed. Windows remains the primary,
  most-verified platform.
- RTT comes from a real ICMP echo per device: `IcmpSendEcho` (iphlpapi) on
  Windows, no admin required; the system `ping` binary elsewhere. A device
  that answers ARP but not ICMP (firewalled) shows online with no RTT.
- The Wi-Fi channel suggestion is a simple congestion count over the visible
  APs, not an RF planner: it does not account for signal strength, DFS
  restrictions, or regulatory rules.
- If this machine has more than one interface up (e.g. Ethernet and Wi-Fi
  connected at once), every one of them is recognized as "This device" by
  MAC, not just the one used to pick the scan's subnet. Otherwise a
  second NIC would show up as an unexplained extra "Computer" with your own
  hostname.
- A device join is flagged "unknown" in the activity log only when its MAC
  has no alias set. Aliasing a device marks it as recognized for future joins.
- When reverse DNS returns nothing, hostname resolution falls back to a
  multicast-DNS reverse lookup and a NetBIOS NBSTAT query, raced against each
  other with a 400ms budget each. Neither is guaranteed: a device with no
  Bonjour/Avahi responder and no NetBIOS support (many phones, most Linux
  boxes without avahi) still shows no hostname. Verified against real
  hardware on the author's LAN, including one device that turned out to
  report "none" as its own mDNS name.
- `netwp ports <ip>` re-probes a single device directly instead of running a
  full network scan: the same well-known TCP ports used for classification,
  reported individually with names, plus a fresh ICMP RTT. There is no
  port-history tracking across runs, just the current state.

## License

[MIT](LICENSE).
