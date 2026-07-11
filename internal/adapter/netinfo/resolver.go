package netinfo

import (
	"context"
	"net"
	"strings"
	"time"
)

// DNSResolver resolves hostnames via reverse DNS. Implements core.HostResolver.
type DNSResolver struct{}

// lookupTimeout caps a single reverse-DNS query. Kept modest because this
// runs before the mDNS/NetBIOS fallback (namelookup.Resolver): a resolver
// that won't answer for RFC1918 reverse zones should fail fast and let the
// fallback try, not stall the whole per-device enrichment for seconds.
const lookupTimeout = 1 * time.Second

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

// Resolve looks up host's A/AAAA records, to confirm forward DNS works.
// Implements core.NameChecker (used by `netwp doctor`).
func (DNSResolver) Resolve(host string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()
	return net.DefaultResolver.LookupIP(ctx, "ip", host)
}
