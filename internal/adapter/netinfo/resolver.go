package netinfo

import (
	"net"
	"strings"
)

// DNSResolver resolves hostnames via reverse DNS. Implements core.HostResolver.
type DNSResolver struct{}

// Hostname returns the first PTR name for ip, or "" if none resolves.
//
// ponytail: net.LookupAddr can stall on hosts with no PTR record. Fine for a
// handful of devices; wrap with a context timeout if scans feel sluggish.
func (DNSResolver) Hostname(ip net.IP) string {
	names, err := net.LookupAddr(ip.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}
