package netinfo

import (
	"net"

	"github.com/gsjonio/netwp/internal/core"
)

// Interface implements core.InterfaceInspector for the active local interface.
type Interface struct{}

func (Interface) Inspect() (core.InterfaceInfo, error) {
	ifi, ipnet, err := activeInterface()
	if err != nil {
		return core.InterfaceInfo{}, err
	}

	var dns []net.IP
	for _, s := range dnsServers(ifi.Name) {
		if ip := net.ParseIP(s); ip != nil {
			dns = append(dns, ip)
		}
	}

	return core.InterfaceInfo{
		Name:       ifi.Name,
		MAC:        ifi.HardwareAddr,
		IP:         ipnet.IP.To4(),
		CIDR:       ipnet,
		Gateway:    DefaultGateway(),
		DNSServers: dns,
		DHCP:       dhcpEnabled(ifi.Name),
	}, nil
}
