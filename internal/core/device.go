package core

import "net"

// Device is a host discovered on the local network.
//
// It is a pure domain type: no I/O, no OS calls. Adapters produce it, the TUI
// consumes it.
type Device struct {
	IP       net.IP           // IPv4 address on the local subnet
	MAC      net.HardwareAddr // Layer-2 address (from ARP)
	Hostname string           // Reverse-DNS name, empty if unresolved
	Vendor   string           // Manufacturer, resolved from the MAC's OUI prefix
	Online   bool             // Answered the active probe during this scan
}
