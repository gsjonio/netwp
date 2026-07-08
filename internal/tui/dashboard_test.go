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
		tracker:  tracker,
		info:     core.InterfaceInfo{Name: "Ethernet", IP: net.IPv4(192, 168, 1, 10), Gateway: net.IPv4(192, 168, 1, 1)},
		start:    time.Now().Add(-90 * time.Second),
		rate:     core.Rate{DownBps: 125000, UpBps: 25000, TotalRx: 1_130_000_000, TotalTx: 243_000_000},
		downHist: []float64{0, 100, 500, 1000, 800, 1200},
		wifiInfo: core.WiFiInfo{Connected: true, SSID: "HomeNet", SignalPercent: 47, Channel: 149, Band: "5 GHz", RxRateMbps: 270, TxRateMbps: 270},
		result:   core.BandwidthResult{DownloadMbps: 2.13, UploadMbps: 6.5},
		speedAt:  time.Now(),
		width:    112,
	}

	out := m.View()
	for _, want := range []string{"netwp dashboard", "WI-FI", "HomeNet", "BANDWIDTH", "Mbps", "SPEEDTEST", "DEVICES", "TV"} {
		if !strings.Contains(out, want) {
			t.Errorf("dashboard view missing %q", want)
		}
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
