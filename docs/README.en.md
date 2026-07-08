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
- [ ] Interface IP configure (static/DHCP)
- [ ] Linux adapter (AF_PACKET raw ARP)

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
.\netwp.exe iface     # active interface's IP config (read-only)
go test ./...
```

The Windows scanner uses the `SendARP` API: **no admin rights and no Npcap
required**.

### Install as `netwp`

`go install` drops the binary in `$(go env GOPATH)\bin`. With that folder on
your PATH you can call it as `netwp` from any terminal (Windows resolves the
`.exe` automatically):

```powershell
go install ./cmd/netwp
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
- The bandwidth test uses Cloudflare's public `speed.cloudflare.com`
  endpoint: no API key, no self-hosted server.
- `iface` is read-only. Changing the address (static/DHCP) needs admin
  rights and isn't implemented yet.

## License

[MIT](../LICENSE).
