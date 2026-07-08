# netwp (English)

**netwp** stands for *Internet / Rede Well Played* ("rede" is Portuguese for network).

A terminal network manager written in Go. It does active local-network device
discovery (ARP); monitoring, bandwidth testing, and interface inspection are
planned. Windows-first, portable to Linux.

🇧🇷 [Versão em português](README.pt-BR.md)

## Status

- [x] Device discovery core (ARP scan, hostname, vendor by OUI, device-class guess)
- [x] Continuous monitoring (join/leave), live TUI
- [x] Bandwidth test
- [x] Interface IP inspect (read-only)
- [x] Interface IP configure (static/DHCP, Windows only)
- [x] Linux adapter (AF_PACKET raw ARP, gateway, DNS)
- [x] Persistent device aliases (nicknames, keyed by MAC)

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
.\netwp.exe            # one-shot scan (default)
.\netwp.exe monitor   # live TUI: devices joining/leaving in real time (q to quit)
.\netwp.exe speedtest # download/upload throughput
.\netwp.exe iface     # active interface's IP config
.\netwp.exe iface static 192.168.1.50/24 192.168.1.1 8.8.8.8  # set a static address (asks to confirm)
.\netwp.exe iface dhcp                                        # switch back to DHCP (asks to confirm)
.\netwp.exe alias set 192.168.1.20 "Living Room TV"  # nickname a device (by IP or MAC)
.\netwp.exe alias ls                                 # list nicknames
.\netwp.exe alias rm 192.168.1.20                    # remove a nickname
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
netwp            # scan
netwp monitor    # live monitor
netwp speedtest  # bandwidth test
netwp iface      # interface IP config
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
- The Linux scanner (raw ARP over `AF_PACKET`) needs `CAP_NET_RAW` (root, or
  `setcap cap_net_raw+ep` on the binary). It was written and cross-compiled
  (`GOOS=linux`) from a Windows dev machine and has not been run against
  real Linux hardware yet.

## License

[MIT](../LICENSE).
