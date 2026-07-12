// Package namelookup resolves a device's hostname when reverse DNS comes up
// empty: most residential devices (phones, TVs, IoT) never register a PTR
// record but do answer multicast DNS or NetBIOS name queries.
package namelookup

import (
	"net"
	"sync"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

const (
	fallbackTimeout = 400 * time.Millisecond
	// hostCacheTTL bounds how stale a cached name may be. The monitor/dashboard
	// re-scans every 10-15s; without a cache every unresolved device pays the
	// full mDNS+NetBIOS race (fallbackTimeout) on every one of those scans.
	// Caching the answer -- including the empty "no name" answer, which is the
	// common, expensive case -- collapses that to one lookup per TTL.
	hostCacheTTL = 5 * time.Minute
)

// Resolver implements core.HostResolver: it asks a primary resolver (reverse
// DNS in production) first, then races mDNS and NetBIOS, first non-empty
// answer wins. Results are cached per IP for hostCacheTTL.
type Resolver struct {
	primary core.HostResolver
	cache   *hostCache
}

// New builds a Resolver whose primary lookup is injected by the composition
// root, so this package depends on the core port rather than another adapter
// and the fallback path can be tested with a fake primary.
func New(primary core.HostResolver) Resolver {
	return Resolver{primary: primary, cache: newHostCache(hostCacheTTL)}
}

// Hostname returns the cached name for ip if still fresh, otherwise resolves it
// (primary, then mDNS/NetBIOS) and caches the result.
func (r Resolver) Hostname(ip net.IP) string {
	key := ip.String()
	if r.cache != nil {
		if name, ok := r.cache.get(key); ok {
			return name
		}
	}
	name := r.resolve(ip)
	if r.cache != nil {
		r.cache.put(key, name)
	}
	return name
}

func (r Resolver) resolve(ip net.IP) string {
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

// hostCache is a small TTL cache of IP-string -> resolved name, safe for the
// concurrent per-device enrichment goroutines.
type hostCache struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]hostEntry
}

type hostEntry struct {
	name    string
	expires time.Time
}

func newHostCache(ttl time.Duration) *hostCache {
	return &hostCache{ttl: ttl, m: map[string]hostEntry{}}
}

func (c *hostCache) get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok || time.Now().After(e.expires) {
		return "", false
	}
	return e.name, true
}

func (c *hostCache) put(key, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = hostEntry{name: name, expires: time.Now().Add(c.ttl)}
}
