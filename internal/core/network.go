package core

import (
	"encoding/binary"
	"net"
)

// Network is the target subnet of a scan: our own address plus its CIDR.
//
// Value object — no behaviour beyond deriving the set of scannable hosts.
type Network struct {
	Self    net.IP     // Our address on this network
	CIDR    *net.IPNet // The subnet (address + mask)
	Gateway net.IP     // Default gateway (the router), nil if undetermined
}

// Hosts returns every usable IPv4 host address in the subnet, excluding the
// network and broadcast addresses.
//
// ponytail: IPv4 only. A /24 yields 254 hosts; huge subnets (/16 = 65k) mean a
// long scan — cap or chunk upstream if that becomes a problem.
func (n Network) Hosts() []net.IP {
	if n.CIDR == nil {
		return nil
	}
	ip4 := n.CIDR.IP.To4()
	mask := net.IP(n.CIDR.Mask).To4()
	if ip4 == nil || mask == nil {
		return nil
	}

	base := binary.BigEndian.Uint32(ip4) & binary.BigEndian.Uint32(mask)
	broadcast := base | ^binary.BigEndian.Uint32(mask)

	hosts := make([]net.IP, 0, broadcast-base)
	for h := base + 1; h < broadcast; h++ {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, h)
		hosts = append(hosts, ip)
	}
	return hosts
}
