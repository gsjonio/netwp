package core

import "net"

// InterfaceInfo describes the active network interface's IP configuration.
type InterfaceInfo struct {
	Name       string
	MAC        net.HardwareAddr
	IP         net.IP
	CIDR       *net.IPNet
	Gateway    net.IP
	DNSServers []net.IP
	DHCP       bool // true if the address was assigned by DHCP, false if static
}

// StaticConfig is a static IPv4 configuration to apply to an interface.
type StaticConfig struct {
	IP      net.IP
	Mask    net.IP // dotted-decimal, e.g. 255.255.255.0
	Gateway net.IP
	DNS     []net.IP
}
