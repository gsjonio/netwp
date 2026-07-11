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

// TestMonitorTruncatesDevicesToFitHeight mirrors
// TestDashboardTruncatesDevicesToFitHeight: netwp monitor never had a height
// budget before, so on a real network with many devices its footer (and the
// r/q key hints) could scroll off a short terminal.
func TestMonitorTruncatesDevicesToFitHeight(t *testing.T) {
	tr := core.NewTracker(30 * time.Second)
	var devices []core.Device
	for i := 0; i < 40; i++ {
		devices = append(devices, core.Device{
			IP:  net.IPv4(192, 168, 1, byte(i+1)),
			MAC: net.HardwareAddr{1, 2, 3, 4, 5, byte(i)},
		})
	}
	tr.Observe(devices, time.Now())

	_, cidr, _ := net.ParseCIDR("192.168.1.0/24")
	m := monitorModel{tracker: tr, network: core.Network{CIDR: cidr}, interval: 10 * time.Second, height: 20}

	out := m.View()
	lines := strings.Split(out, "\n")
	if len(lines) > m.height+1 { // +1 tolerance: line budget estimate, not pixel-exact
		t.Errorf("rendered %d lines, want <= ~%d (height=%d)", len(lines), m.height+1, m.height)
	}
	if !strings.Contains(out, "r rescan") {
		t.Error("footer missing from a short-terminal render")
	}
	if !strings.Contains(out, "showing") {
		t.Error("expected the summary line to note truncation")
	}
}

func TestTruncateToHeight(t *testing.T) {
	var devices []core.TrackedDevice
	for i := 0; i < 10; i++ {
		devices = append(devices, core.TrackedDevice{})
	}
	if shown, truncated := truncateToHeight(devices, 0); truncated || len(shown) != 10 {
		t.Errorf("budget<=0: got %d devices, truncated=%v, want all 10 unchanged", len(shown), truncated)
	}
	if shown, truncated := truncateToHeight(devices, 20); truncated || len(shown) != 10 {
		t.Errorf("budget > total: got %d devices, truncated=%v, want all 10 unchanged", len(shown), truncated)
	}
	if shown, truncated := truncateToHeight(devices, 4); !truncated || len(shown) != 4 {
		t.Errorf("budget=4: got %d devices, truncated=%v, want 4 and truncated=true", len(shown), truncated)
	}
}

// TestBandwidthLine cannot assert color (see TestPortsCellText's comment on
// why), only text/absence: disabled when reader is nil, present with rates
// when set, and mentioning the threshold once it's crossed.
func TestBandwidthLine(t *testing.T) {
	m := monitorModel{}
	if got := m.bandwidthLine(); got != "" {
		t.Errorf("bandwidthLine() with nil reader = %q, want empty", got)
	}

	m = monitorModel{reader: fakeCounterReader{}, rate: core.Rate{DownBps: 2_000_000, UpBps: 500_000}}
	if got := m.bandwidthLine(); !strings.Contains(got, "Mbps") {
		t.Errorf("bandwidthLine() = %q, want it to mention Mbps", got)
	}

	m.alertDown = 5_000_000 // above the 2,000,000 DownBps above: alert should fire
	if got := m.bandwidthLine(); !strings.Contains(got, "below") {
		t.Errorf("bandwidthLine() over threshold = %q, want it to mention the alert", got)
	}
}

type fakeCounterReader struct{}

func (fakeCounterReader) Counters() (core.NetCounters, error) { return core.NetCounters{}, nil }

type fakeEventLogger struct{ events []core.Event }

func (f *fakeEventLogger) Log(e core.Event) error {
	f.events = append(f.events, e)
	return nil
}

// TestMonitorLogsEventsToLogger checks scanDoneMsg forwards every tracker
// event to the logger, when one is set (nil logger is the default, tested
// implicitly by every other test in this file not setting one and not panicking).
func TestMonitorLogsEventsToLogger(t *testing.T) {
	tr := core.NewTracker(30 * time.Second)
	logger := &fakeEventLogger{}
	_, cidr, _ := net.ParseCIDR("192.168.1.0/24")
	m := monitorModel{tracker: tr, network: core.Network{CIDR: cidr}, interval: 10 * time.Second, logger: logger}

	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	m.Update(scanDoneMsg{devices: []core.Device{{IP: net.IPv4(192, 168, 1, 5), MAC: mac}}, at: time.Now()})

	if len(logger.events) != 1 || logger.events[0].Kind != core.Joined {
		t.Errorf("logger.events = %+v, want one Joined event", logger.events)
	}
}

func TestIsAlertEvent(t *testing.T) {
	unknownJoin := core.Event{Kind: core.Joined, Device: core.Device{Alias: ""}}
	knownJoin := core.Event{Kind: core.Joined, Device: core.Device{Alias: "Meu PC"}}
	leave := core.Event{Kind: core.Left, Device: core.Device{Alias: "Câmera"}}

	if !isAlertEvent(unknownJoin, false) {
		t.Error("an unknown device joining should alert")
	}
	if isAlertEvent(knownJoin, false) {
		t.Error("a known (aliased) device joining should not alert")
	}
	if !isAlertEvent(leave, true) {
		t.Error("a watched device leaving should alert")
	}
	if isAlertEvent(leave, false) {
		t.Error("an unwatched device leaving should not alert")
	}
}

func TestFormatEventWatchedLeft(t *testing.T) {
	e := core.Event{Kind: core.Left, Device: core.Device{Alias: "Câmera"}, At: time.Now()}
	if got := formatEvent(e, true); !strings.Contains(got, "watched") {
		t.Errorf("formatEvent(watched left) = %q, want it to mention \"watched\"", got)
	}
	if got := formatEvent(e, false); strings.Contains(got, "watched") {
		t.Errorf("formatEvent(unwatched left) = %q, should not mention \"watched\"", got)
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
