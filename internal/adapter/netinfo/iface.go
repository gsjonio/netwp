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
	return core.Network{
		Self:      ipnet.IP.To4(),
		CIDR:      ipnet,
		Gateway:   DefaultGateway(),
		LocalMACs: localMACs(),
	}, nil
}

// localMACs collects the hardware address of every up, non-loopback
// interface on this machine, not just the one activeInterface picked for
// Self/CIDR. A machine with both Ethernet and Wi-Fi connected at once shows
// up as two separate devices in a scan; this lets Classify recognize the
// second one as "this device" too, by MAC instead of by IP.
func localMACs() []net.HardwareAddr {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var macs []net.HardwareAddr
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(ifi.HardwareAddr) > 0 {
			macs = append(macs, ifi.HardwareAddr)
		}
	}
	return macs
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
