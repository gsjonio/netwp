// Package namelookup resolves a device's hostname when reverse DNS comes up
// empty: most residential devices (phones, TVs, IoT) never register a PTR
// record but do answer multicast DNS or NetBIOS name queries.
package namelookup

import (
	"net"
	"time"

	"github.com/gsjonio/netwp/internal/adapter/netinfo"
)

const fallbackTimeout = 400 * time.Millisecond

// Resolver implements core.HostResolver: reverse DNS first, then mDNS and
// NetBIOS raced against each other, first non-empty answer wins.
type Resolver struct {
	dns netinfo.DNSResolver
}

func New() Resolver { return Resolver{} }

// ponytail: no result caching, no config for the fallback timeout. Every
// unresolved device pays up to fallbackTimeout on every scan; add a
// short-lived cache if repeat scans get noticeably slower on large LANs.
func (r Resolver) Hostname(ip net.IP) string {
	if name := r.dns.Hostname(ip); name != "" {
		return name
	}

	out := make(chan string, 2)
	go func() { out <- mdnsReverseLookup(ip, fallbackTimeout) }()
	go func() { out <- netbiosLookup(ip, fallbackTimeout) }()

	for i := 0; i < 2; i++ {
		if name := <-out; name != "" {
			return name
		}
	}
	return ""
}
