package namelookup

import (
	"net"
	"testing"
	"time"
)

type fakePrimary struct{ name string }

func (f fakePrimary) Hostname(net.IP) string { return f.name }

// countingPrimary records how many times it was asked, to prove the cache
// spares the resolver from re-querying the same IP within the TTL.
type countingPrimary struct {
	name  string
	calls int
}

func (c *countingPrimary) Hostname(net.IP) string {
	c.calls++
	return c.name
}

// TestResolverPrimaryShortCircuits proves a primary hit is returned without
// falling through to the mDNS/NetBIOS network path. Injecting the primary is
// the point of the DIP refactor: this test needs no network. (The fallback
// path itself hits the network, so it is left to integration, not unit,
// coverage — the wire parsing it relies on is covered in dnswire_test.go.)
func TestResolverPrimaryShortCircuits(t *testing.T) {
	r := New(fakePrimary{name: "router.local"})
	if got := r.Hostname(net.IPv4(192, 168, 1, 1)); got != "router.local" {
		t.Errorf("Hostname = %q, want router.local (primary should short-circuit)", got)
	}
}

// TestResolverCachesResult proves a second lookup of the same IP within the TTL
// is served from cache, not by re-querying the primary (nor the network).
func TestResolverCachesResult(t *testing.T) {
	p := &countingPrimary{name: "host.local"}
	r := New(p)
	ip := net.IPv4(192, 168, 1, 42)

	if got := r.Hostname(ip); got != "host.local" {
		t.Fatalf("first lookup = %q, want host.local", got)
	}
	if got := r.Hostname(ip); got != "host.local" {
		t.Fatalf("second lookup = %q, want host.local", got)
	}
	if p.calls != 1 {
		t.Errorf("primary queried %d times, want 1 (second lookup should hit the cache)", p.calls)
	}
}

// TestResolverResetCache proves a manual rescan re-queries the primary: after
// ResetCache, a previously cached IP is looked up again instead of served stale.
func TestResolverResetCache(t *testing.T) {
	p := &countingPrimary{name: "host.local"}
	r := New(p)
	ip := net.IPv4(192, 168, 1, 42)

	r.Hostname(ip) // populates the cache (calls == 1)
	r.ResetCache() // a manual rescan drops it
	r.Hostname(ip) // must re-query, not serve the cache
	if p.calls != 2 {
		t.Errorf("primary queried %d times, want 2 (ResetCache should force a re-query)", p.calls)
	}
}

func TestHostCacheExpiry(t *testing.T) {
	c := newHostCache(time.Minute)
	c.put("10.0.0.1", "a.local")
	if got, ok := c.get("10.0.0.1"); !ok || got != "a.local" {
		t.Errorf("fresh entry: got (%q, %v), want (a.local, true)", got, ok)
	}
	// Force an expired entry and confirm it misses.
	c.m["10.0.0.2"] = hostEntry{name: "b.local", expires: time.Now().Add(-time.Second)}
	if _, ok := c.get("10.0.0.2"); ok {
		t.Error("expired entry should miss")
	}
	// A cached empty name (the common, expensive case) is still a hit.
	c.put("10.0.0.3", "")
	if got, ok := c.get("10.0.0.3"); !ok || got != "" {
		t.Errorf("cached empty name: got (%q, %v), want (\"\", true)", got, ok)
	}
}
