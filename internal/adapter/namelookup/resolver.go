// Package namelookup resolves a device's hostname when reverse DNS comes up
// empty: most residential devices (phones, TVs, IoT) never register a PTR
// record but do answer multicast DNS or NetBIOS name queries.
package namelookup

import (
	"net"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

const fallbackTimeout = 400 * time.Millisecond

// Resolver implements core.HostResolver: it asks a primary resolver (reverse
// DNS in production) first, then races mDNS and NetBIOS, first non-empty
// answer wins.
type Resolver struct {
	primary core.HostResolver
}

// New builds a Resolver whose primary lookup is injected by the composition
// root, so this package depends on the core port rather than another adapter
// and the fallback path can be tested with a fake primary.
func New(primary core.HostResolver) Resolver { return Resolver{primary: primary} }

// ponytail: no result caching, no config for the fallback timeout. Every
// unresolved device pays up to fallbackTimeout on every scan; add a
// short-lived cache if repeat scans get noticeably slower on large LANs.
func (r Resolver) Hostname(ip net.IP) string {
	if r.primary != nil {
		if name := r.primary.Hostname(ip); name != "" {
			return name
		}
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
