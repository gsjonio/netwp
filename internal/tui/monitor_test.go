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
