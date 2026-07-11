package tui

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

// TestDashboardViewSmoke renders a populated model with no TTY and checks the
// panels appear, guarding against layout/format panics.
func TestDashboardViewSmoke(t *testing.T) {
	tracker := core.NewTracker(30 * time.Second)
	tracker.Observe([]core.Device{
		{IP: net.IPv4(192, 168, 1, 5), MAC: net.HardwareAddr{1, 2, 3, 4, 5, 6}, Alias: "TV", Online: true},
	}, time.Now())

	m := dashModel{
		tracker:   tracker,
		info:      core.InterfaceInfo{Name: "Ethernet", IP: net.IPv4(192, 168, 1, 10), Gateway: net.IPv4(192, 168, 1, 1)},
		start:     time.Now().Add(-90 * time.Second),
		rate:      core.Rate{DownBps: 125000, UpBps: 25000, TotalRx: 1_130_000_000, TotalTx: 243_000_000},
		downHist:  []float64{0, 100, 500, 1000, 800, 1200},
		wifiInfo:  core.WiFiInfo{Connected: true, SSID: "HomeNet", SignalPercent: 47, Channel: 149, Band: "5 GHz", RxRateMbps: 270, TxRateMbps: 270},
		wifiHist:  []float64{40, 42, 45, 47},
		result:    core.BandwidthResult{DownloadMbps: 2.13, UploadMbps: 6.5},
		speedHist: []float64{1.8, 2.0, 2.13},
		speedAt:   time.Now(),
		netUp:     true,
		netHist:   []float64{8, 10, 9, 11},
		width:     112,
	}

	out := m.View()
	for _, want := range []string{
		"netwp dashboard", "WI-FI", "HomeNet", "BANDWIDTH", "Mbps", "SPEEDTEST", "DEVICES", "TV",
		"8-11ms", "40-47%", "1.8-2.1 Mbps", // sparkline range labels
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dashboard view missing %q", want)
		}
	}
}

// TestDashboardTruncatesDevicesToFitHeight builds a model with far more
// devices than a short terminal can show, and checks the footer still
// appears within the reported height (it must not scroll off screen) and
// that the device list was actually shortened.
func TestDashboardTruncatesDevicesToFitHeight(t *testing.T) {
	tracker := core.NewTracker(30 * time.Second)
	var devices []core.Device
	for i := 0; i < 40; i++ {
		devices = append(devices, core.Device{
			IP:  net.IPv4(192, 168, 1, byte(i+1)),
			MAC: net.HardwareAddr{1, 2, 3, 4, 5, byte(i)},
		})
	}
	tracker.Observe(devices, time.Now())

	m := dashModel{
		tracker: tracker,
		info:    core.InterfaceInfo{Name: "Ethernet", IP: net.IPv4(192, 168, 1, 10), Gateway: net.IPv4(192, 168, 1, 1)},
		start:   time.Now(),
		width:   112,
		height:  20,
	}

	out := m.View()
	lines := strings.Split(out, "\n")
	if len(lines) > m.height+1 { // +1 tolerance: this is a line budget estimate, not pixel-exact
		t.Errorf("rendered %d lines, want <= ~%d (height=%d)", len(lines), m.height+1, m.height)
	}
	if !strings.Contains(out, "r rescan") {
		t.Error("footer missing from a short-terminal render")
	}
	if !strings.Contains(out, "showing") {
		t.Error("expected the device panel title to note truncation")
	}
}

// TestDashboardLogPanel checks the LOG panel renders its operation lines on a
// terminal with room, without pushing the footer off screen.
func TestDashboardLogPanel(t *testing.T) {
	m := dashModel{
		tracker: core.NewTracker(30 * time.Second),
		info:    core.InterfaceInfo{Name: "Ethernet", IP: net.IPv4(192, 168, 1, 10)},
		start:   time.Now(),
		width:   112,
		height:  40,
		ops:     []string{opLine("running scan…"), opLine("scan done · 5 devices")},
	}
	out := m.View()
	if !strings.Contains(out, "LOG") {
		t.Error("LOG panel title missing")
	}
	if !strings.Contains(out, "scan done · 5 devices") {
		t.Error("expected the latest operation line in the LOG panel")
	}
	if !strings.Contains(out, "r rescan") {
		t.Error("footer missing")
	}
}

// TestDashboardLogsScanLifecycle checks the scan start and completion each add
// an operation-log line via Update.
func TestDashboardLogsScanLifecycle(t *testing.T) {
	m := dashModel{tracker: core.NewTracker(30 * time.Second), start: time.Now()}

	u, _ := m.Update(scanTickMsg{})
	m = u.(dashModel)
	if !strings.Contains(strings.Join(m.ops, "\n"), "running scan") {
		t.Errorf("scanTickMsg should log a 'running scan' line, got %v", m.ops)
	}

	u, _ = m.Update(scanMsg{devices: []core.Device{{IP: net.IPv4(10, 0, 0, 1), MAC: net.HardwareAddr{1, 2, 3, 4, 5, 6}}}, at: time.Now()})
	m = u.(dashModel)
	if !strings.Contains(strings.Join(m.ops, "\n"), "scan done · 1 devices") {
		t.Errorf("scanMsg should log a 'scan done' line, got %v", m.ops)
	}
}

func TestClassSummary(t *testing.T) {
	devices := []core.TrackedDevice{
		{Device: core.Device{Class: core.ClassRouter}, Online: true},
		{Device: core.Device{Class: core.ClassMedia}, Online: true},
		{Device: core.Device{Class: core.ClassMedia}, Online: true},
		{Device: core.Device{Class: core.ClassThisDevice}, Online: true}, // skipped
		{Device: core.Device{Class: core.ClassUnknown}, Online: true},    // skipped
		{Device: core.Device{Class: core.ClassIoT}, Online: false},       // offline: not counted
	}
	got := classSummary(devices)
	if want := "2 Media · 1 Router"; got != want {
		t.Errorf("classSummary() = %q, want %q", got, want)
	}
}

func TestClassSummaryEmpty(t *testing.T) {
	devices := []core.TrackedDevice{
		{Device: core.Device{Class: core.ClassThisDevice}, Online: true},
		{Device: core.Device{Class: core.ClassUnknown}, Online: true},
	}
	if got := classSummary(devices); got != "" {
		t.Errorf("classSummary() = %q, want empty (only skippable classes online)", got)
	}
}

// TestDashboardStacksPanelsOnNarrowTerminal checks the three top panels
// (WI-FI/BANDWIDTH/SPEEDTEST) stack vertically instead of sitting side by
// side once the terminal is too narrow for three ~22-column panels --
// below dashNarrowCols they used to wrap mid-word instead.
func TestDashboardStacksPanelsOnNarrowTerminal(t *testing.T) {
	m := dashModel{tracker: core.NewTracker(30 * time.Second), width: 70}
	out := m.View()

	// Stacked: each panel's own top border line appears on its own line,
	// nothing to its right. Side-by-side output would instead show all
	// three top borders concatenated on a single line.
	lines := strings.Split(out, "\n")
	borders := 0
	for _, line := range lines {
		if strings.Count(line, "╭") == 1 && strings.Count(line, "╮") == 1 {
			borders++
		}
	}
	if borders < 3 {
		t.Errorf("expected at least 3 lines with exactly one panel's top border (stacked layout), got %d\n%s", borders, out)
	}
	for _, want := range []string{"WI-FI", "BANDWIDTH", "SPEEDTEST"} {
		if !strings.Contains(out, want) {
			t.Errorf("narrow view missing panel %q", want)
		}
	}
}

// TestDashboardKeepsPanelsSideBySideWhenWide is the contrast case: at or
// above dashNarrowCols, the three top panels' borders must land on the same
// line, confirming the width check actually branches both ways.
func TestDashboardKeepsPanelsSideBySideWhenWide(t *testing.T) {
	m := dashModel{tracker: core.NewTracker(30 * time.Second), width: dashDefaultCols}
	out := m.View()
	for _, line := range strings.Split(out, "\n") {
		if strings.Count(line, "╭") == 3 {
			return // found the joined top-border line -- side by side, as expected
		}
	}
	t.Errorf("expected a line with all three panels' top borders side by side (wide layout), got:\n%s", out)
}

func TestSparklineRange(t *testing.T) {
	if _, _, ok := sparklineRange(nil); ok {
		t.Error("sparklineRange(nil) ok = true, want false")
	}
	lo, hi, ok := sparklineRange([]float64{5, 1, 9, 3})
	if !ok || lo != 1 || hi != 9 {
		t.Errorf("sparklineRange([5,1,9,3]) = (%v, %v, %v), want (1, 9, true)", lo, hi, ok)
	}
	lo, hi, ok = sparklineRange([]float64{4})
	if !ok || lo != 4 || hi != 4 {
		t.Errorf("sparklineRange([4]) = (%v, %v, %v), want (4, 4, true)", lo, hi, ok)
	}
}

func TestPushHistTrimsToLimit(t *testing.T) {
	var hist []float64
	for i := 0; i < histLen+10; i++ {
		hist = pushHist(hist, float64(i))
	}
	if len(hist) != histLen {
		t.Fatalf("len(hist) = %d, want %d", len(hist), histLen)
	}
	if want := float64(histLen + 9); hist[len(hist)-1] != want {
		t.Errorf("last sample = %v, want %v (oldest samples should be dropped, not newest)", hist[len(hist)-1], want)
	}
}

func TestRateStr(t *testing.T) {
	cases := map[float64]string{
		125000: "1.0 Mbps", // 125000 B/s * 8 = 1 Mbps
		1000:   "8.0 Kbps",
		10:     "80 bps",
	}
	for bps, want := range cases {
		if got := rateStr(bps); got != want {
			t.Errorf("rateStr(%v) = %q, want %q", bps, got, want)
		}
	}
}
