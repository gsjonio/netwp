package core

import (
	"context"
	"net"
)

// Scanner performs active discovery of hosts on a target network.
//
// This is the central port. Implementations are platform-specific and selected
// at build time (Windows: SendARP; Linux: raw ARP over AF_PACKET). The core
// never knows which one it talks to.
type Scanner interface {
	Scan(ctx context.Context, target Network) ([]Device, error)
}

// HostResolver turns an IP into a hostname (reverse DNS). Cross-platform.
type HostResolver interface {
	Hostname(ip net.IP) string
}

// VendorLookup resolves a MAC address to a manufacturer via its OUI prefix.
type VendorLookup interface {
	Vendor(mac net.HardwareAddr) string
}

// Prober reports which of a small set of well-known TCP ports a host accepts
// connections on — the "detailed scan" used to refine device classification.
type Prober interface {
	OpenPorts(ctx context.Context, ip net.IP) []int
}

// InterfaceInspector reads the active network interface's IP configuration.
type InterfaceInspector interface {
	Inspect() (InterfaceInfo, error)
}
