//go:build !windows && !linux

package netinfo

import "net"

// DefaultGateway is not implemented off Windows yet; classification simply skips
// the router hint until the Linux adapter lands.
//
// ponytail: stub returns nil so the cross-platform code compiles. Implement via
// netlink (parse /proc/net/route or RTM_GETROUTE) with the Linux port.
func DefaultGateway() net.IP { return nil }
