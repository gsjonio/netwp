package namelookup

import (
	"net"
	"testing"
)

type fakePrimary struct{ name string }

func (f fakePrimary) Hostname(net.IP) string { return f.name }

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
