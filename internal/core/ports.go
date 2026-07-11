package core

import (
	"context"
	"net"
	"time"
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

// AliasLookup returns a user-defined nickname for a MAC, or "" if none is set.
type AliasLookup interface {
	Alias(mac net.HardwareAddr) string
}

// ClassLookup returns a user-pinned device class for a MAC, overriding the
// automatic guess. ok is false when the user hasn't pinned one.
type ClassLookup interface {
	ClassOverride(mac net.HardwareAddr) (DeviceClass, bool)
}

// Prober reports which of a small set of well-known TCP ports a host accepts
// connections on — the "detailed scan" used to refine device classification.
type Prober interface {
	OpenPorts(ctx context.Context, ip net.IP) []int
}

// Pinger measures ICMP round-trip time (and TTL, when the platform reports
// one) to a host. ok is false on timeout or error (host unreachable), in
// which case rtt and ttl are meaningless. ttl is 0 when unavailable.
type Pinger interface {
	Ping(ip net.IP, timeout time.Duration) (rtt time.Duration, ttl int, ok bool)
}

// EventLogger persists a presence-change Event for later review (`netwp
// events`). Best-effort: callers ignore its error, the same way scancache
// writes are best-effort.
type EventLogger interface {
	Log(e Event) error
}

// InterfaceInspector reads the active network interface's IP configuration.
type InterfaceInspector interface {
	Inspect() (InterfaceInfo, error)
}

// InterfaceConfigurator applies an IPv4 configuration change to the active
// interface. Implementations require elevated/admin privileges.
type InterfaceConfigurator interface {
	SetStatic(cfg StaticConfig) error
	SetDHCP() error
}
