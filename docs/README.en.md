# netwp — English

A terminal network manager written in Go. Active local-network device discovery
(ARP), with monitoring, bandwidth testing and interface inspection planned.
Windows-first, portable to Linux.

🇧🇷 [Versão em português](README.pt-BR.md)

## Status

- [x] Device discovery core (ARP scan, hostname, vendor by OUI)
- [x] Continuous monitoring (join/leave), live TUI
- [ ] Bandwidth test
- [ ] Interface IP inspect/configure
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
go test ./...
```

The Windows scanner uses the `SendARP` API: **no admin rights and no Npcap
required**.

## Notes

- Vendor names come from the full IEEE MA-L registry, gzipped and embedded in
  the binary (`internal/adapter/oui/data`). Refresh it with the command in
  `oui.go`.
- Active scanning may be flagged as intrusive on managed/corporate networks.
  Only scan networks you own or are authorized to.

## License

[MIT](../LICENSE).
