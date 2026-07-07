package core

import (
	"net"
	"testing"
)

func mustCIDR(t *testing.T, s string) Network {
	t.Helper()
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return Network{CIDR: ipnet}
}

func TestHosts(t *testing.T) {
	cases := []struct {
		cidr      string
		wantCount int
		wantFirst string
		wantLast  string
	}{
		{"192.168.0.0/30", 2, "192.168.0.1", "192.168.0.2"}, // 4 addrs - net - bcast
		{"10.0.0.0/24", 254, "10.0.0.1", "10.0.0.254"},      // classic /24
		{"192.168.1.0/31", 0, "", ""},                       // no usable hosts
		{"192.168.1.5/32", 0, "", ""},                       // single address
	}

	for _, c := range cases {
		hosts := mustCIDR(t, c.cidr).Hosts()
		if len(hosts) != c.wantCount {
			t.Errorf("%s: got %d hosts, want %d", c.cidr, len(hosts), c.wantCount)
			continue
		}
		if c.wantCount == 0 {
			continue
		}
		if got := hosts[0].String(); got != c.wantFirst {
			t.Errorf("%s: first host = %s, want %s", c.cidr, got, c.wantFirst)
		}
		if got := hosts[len(hosts)-1].String(); got != c.wantLast {
			t.Errorf("%s: last host = %s, want %s", c.cidr, got, c.wantLast)
		}
	}
}
