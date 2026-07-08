package core

import (
	"net"
	"time"
)

// Device is a host discovered on the local network.
//
// It is a pure domain type: no I/O, no OS calls. Adapters produce it, the TUI
// consumes it.
type Device struct {
	IP        net.IP           // IPv4 address on the local subnet
	MAC       net.HardwareAddr // Layer-2 address (from ARP)
	Alias     string           // User-defined nickname, keyed by MAC; empty if unset
	Hostname  string           // Reverse-DNS name, empty if unresolved
	Vendor    string           // Manufacturer, resolved from the MAC's OUI prefix
	Class     DeviceClass      // Best-effort guess at what kind of device this is
	RTT       time.Duration    // ICMP round-trip time; 0 with Reachable false if no reply
	Reachable bool             // Answered an ICMP echo during enrichment
	Online    bool             // Answered the active probe during this scan
}
