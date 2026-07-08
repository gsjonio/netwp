// Package netinfo reads local network interface information. Pure stdlib, so it
// is cross-platform (Windows and Linux) with no build tags.
package netinfo

import (
	"errors"
	"net"

	"github.com/gsjonio/netwp/internal/core"
)

// LocalNetwork returns the first active IPv4 interface and its subnet.
//
// ponytail: picks the first up, non-loopback IPv4 interface. Multi-homed hosts
// (VPN, Wi-Fi + Ethernet) may need an explicit selection flag later.
func LocalNetwork() (core.Network, error) {
	_, ipnet, err := activeInterface()
	if err != nil {
		return core.Network{}, err
	}
	return core.Network{Self: ipnet.IP.To4(), CIDR: ipnet, Gateway: DefaultGateway()}, nil
}

// activeInterface returns the first active, non-loopback interface carrying
// an IPv4 address, and that address's network.
func activeInterface() (net.Interface, *net.IPNet, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return net.Interface{}, nil, err
	}
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}
			return ifi, ipnet, nil
		}
	}
	return net.Interface{}, nil, errors.New("no active IPv4 interface found")
}
