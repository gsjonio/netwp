# Security Policy

## Reporting a vulnerability

Please report security issues privately via
[GitHub Security Advisories](https://github.com/gsjonio/netwp/security/advisories/new)
for this repository, instead of opening a public issue. You'll get a
response as soon as possible.

## Supported versions

Only the latest tagged release is supported. There is no long-term support
branch.

## Things to know before using netwp

netwp is a network reconnaissance and configuration tool. A few of its
features are inherently sensitive:

- **Active scanning.** `netwp scan`, `monitor`, and `dashboard` send ARP
  requests to every host on your subnet. This is normal on a home network
  but can look like a probe on a managed or corporate network, and may
  violate its acceptable-use policy. Only scan networks you own or are
  explicitly authorized to scan.
- **Real network configuration changes.** `netwp iface static`/`iface dhcp`
  change the active interface's real IP configuration on Windows. Both
  require an elevated (admin) terminal and always ask for a typed "yes"
  before touching anything; there is no `--yes` flag to skip that
  confirmation.
- **Local data only.** Device aliases are stored as plain-text JSON in
  `<user-config-dir>/netwp/aliases.json`. Nothing is sent off your machine
  except the scan/probe traffic itself (ARP, ICMP, TCP connect probes to a
  small set of well-known ports) and the Wi-Fi/Cloudflare/mDNS/NetBIOS
  lookups documented in the [README](README.md#notes). No telemetry, no
  external service receives your scan results.

## Dependencies

netwp depends only on the Go standard library plus
[bubbletea](https://github.com/charmbracelet/bubbletea) and
[lipgloss](https://github.com/charmbracelet/lipgloss) for the terminal UI.
Dependency updates are reviewed manually; there's no automated dependency
bot configured yet.
