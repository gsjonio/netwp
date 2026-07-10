package tui

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

// Smoke test the render path without a TTY: it exercises the lipgloss table and
// StyleFunc row indexing, which would panic if the row/device mapping is wrong.
func TestMonitorViewSmoke(t *testing.T) {
	tr := core.NewTracker(30 * time.Second)
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	tr.Observe([]core.Device{
		{IP: net.ParseIP("192.168.0.5").To4(), MAC: mac, Hostname: "host", Vendor: "Acme", Online: true},
	}, time.Unix(0, 0))

	_, cidr, _ := net.ParseCIDR("192.168.0.0/24")
	m := monitorModel{tracker: tr, network: core.Network{CIDR: cidr}, interval: 10 * time.Second}

	out := m.View()
	for _, want := range []string{"netwp monitor", "192.168.0.5", "Acme", "q quit"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q\n---\n%s", want, out)
		}
	}
}

func TestHasSensitivePort(t *testing.T) {
	cases := []struct {
		ports []int
		want  bool
	}{
		{nil, false},
		{[]int{80, 443}, false},
		{[]int{22}, true},
		{[]int{445}, true},
		{[]int{3389}, true},
		{[]int{80, 3389, 8009}, true},
	}
	for _, c := range cases {
		if got := hasSensitivePort(c.ports); got != c.want {
			t.Errorf("hasSensitivePort(%v) = %v, want %v", c.ports, got, c.want)
		}
	}
}

// TestPortsCellText only checks the text survives the styWarn wrapping
// intact. It cannot assert that a sensitive port is actually rendered in
// color: lipgloss disables color output outside a real terminal, so
// styWarn.Render is a no-op passthrough under `go test` regardless of input
// (same reason aliasText/styAlias has no color assertion in this package).
func TestPortsCellText(t *testing.T) {
	if got := portsCellText([]int{80, 443}); !strings.Contains(got, "80,443") {
		t.Errorf("portsCellText([80,443]) = %q, want it to contain \"80,443\"", got)
	}
	if got := portsCellText([]int{22}); !strings.Contains(got, "22") {
		t.Errorf("portsCellText([22]) = %q, want it to contain \"22\"", got)
	}
}
