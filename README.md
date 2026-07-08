# netwp

🇧🇷 [Português](README.pt-BR.md)

**netwp** stands for *Internet / Rede Well Played* ("rede" is Portuguese for network).

A terminal network manager written in Go: active local-network device discovery
(ARP), live monitoring, a full dashboard, bandwidth testing, and interface
inspection. Windows-first, portable to Linux.

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

## Build & Run

Requires Go 1.22+.

```powershell
go build -o netwp.exe ./cmd/netwp
.\netwp.exe            # one-shot scan (default), with per-device RTT
.\netwp.exe scan --json # same scan, machine-readable JSON on stdout
.\netwp.exe monitor   # live TUI: devices joining/leaving in real time (q to quit)
.\netwp.exe dashboard # full dashboard: wifi + live bandwidth + speedtest + devices
.\netwp.exe speedtest # download/upload throughput
.\netwp.exe iface     # active interface's IP config
.\netwp.exe iface static 192.168.1.50/24 192.168.1.1 8.8.8.8  # set a static address (asks to confirm)
.\netwp.exe iface dhcp                                        # switch back to DHCP (asks to confirm)
.\netwp.exe alias set 192.168.1.20 "Living Room TV"  # nickname a device (by IP or MAC)
.\netwp.exe alias ls                                 # list nicknames
.\netwp.exe alias rm 192.168.1.20                    # remove a nickname
.\netwp.exe ports 192.168.1.20                       # open ports + RTT for one device
go test ./...
```

For a smaller binary, strip the symbol table and DWARF info
(about 12 MB down to 8.8 MB):

```powershell
go build -ldflags "-s -w" -o netwp.exe ./cmd/netwp
```

The Windows scanner uses the `SendARP` API: **no admin rights and no Npcap
required**.

### Install as `netwp`

`go install` drops the binary in `$(go env GOPATH)\bin`. With that folder on
your PATH you can call it as `netwp` from any terminal (Windows resolves the
`.exe` automatically):

```powershell
go install -ldflags "-s -w" ./cmd/netwp   # -ldflags optional, just for a smaller binary
netwp             # scan
netwp scan --json # scan, JSON output
netwp monitor     # live monitor
netwp dashboard   # full dashboard
netwp speedtest   # bandwidth test
netwp iface       # interface IP config
netwp alias set 192.168.1.20 "Living Room TV"  # nickname a device
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
  endpoint: no API key, no self-hosted server.
- `iface static`/`iface dhcp` shell out to `netsh` and need an elevated
  (admin) terminal on Windows. They always ask for a typed "yes" before
  touching the real configuration; there's no `--yes` flag to skip it.
  Not implemented on Linux yet.
- The dashboard's Wi-Fi panel reads `netsh wlan` (English and Portuguese
  labels supported). Verified on real hardware in both states: disconnected,
  and connected (SSID/BSSID/channel/signal/Rx-Tx rate of your own
  association). The English labels are still fixture-only, since testing
  them needs an English-locale Windows install. On a wired-only host the
  panel shows "disconnected".
- The Linux scanner (raw ARP over `AF_PACKET`) needs `CAP_NET_RAW` (root, or
  `setcap cap_net_raw+ep` on the binary). It was written and cross-compiled
  (`GOOS=linux`) from a Windows dev machine and has not been run against
  real Linux hardware yet.
- RTT comes from a real ICMP echo per device: `IcmpSendEcho` (iphlpapi) on
  Windows, no admin required; the system `ping` binary elsewhere. A device
  that answers ARP but not ICMP (firewalled) shows online with no RTT.
- The Wi-Fi channel suggestion is a simple congestion count over the visible
  APs, not an RF planner: it does not account for signal strength, DFS
  restrictions, or regulatory rules.
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
