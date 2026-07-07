package netinfo

import (
	"context"
	"net"
	"strings"
	"time"
)

// DNSResolver resolves hostnames via reverse DNS. Implements core.HostResolver.
type DNSResolver struct{}

const lookupTimeout = 2 * time.Second

// Hostname returns the first PTR name for ip, or "" if none resolves.
//
// Bounded to lookupTimeout so a host without a PTR record cannot stall a scan.
func (DNSResolver) Hostname(ip net.IP) string {
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(ctx, ip.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}
