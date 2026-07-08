package core

import (
	"testing"
	"time"
)

func TestRateMeter(t *testing.T) {
	var m RateMeter
	t0 := time.Now()

	// First sample: baseline, zero rates.
	r := m.Update(NetCounters{RxBytes: 1000, TxBytes: 500}, t0)
	if r.DownBps != 0 || r.UpBps != 0 {
		t.Errorf("first sample should be zero rate, got %+v", r)
	}

	// 2s later: +2000 rx, +1000 tx -> 1000 B/s down, 500 B/s up.
	r = m.Update(NetCounters{RxBytes: 3000, TxBytes: 1500}, t0.Add(2*time.Second))
	if r.DownBps != 1000 {
		t.Errorf("DownBps = %v, want 1000", r.DownBps)
	}
	if r.UpBps != 500 {
		t.Errorf("UpBps = %v, want 500", r.UpBps)
	}
	if r.TotalRx != 2000 || r.TotalTx != 1000 {
		t.Errorf("totals = %d/%d, want 2000/1000", r.TotalRx, r.TotalTx)
	}
}

func TestRateMeterCounterReset(t *testing.T) {
	var m RateMeter
	t0 := time.Now()
	m.Update(NetCounters{RxBytes: 5000, TxBytes: 5000}, t0)

	// Counter goes backwards (NIC reset): rate must clamp to 0, not spike.
	r := m.Update(NetCounters{RxBytes: 100, TxBytes: 100}, t0.Add(time.Second))
	if r.DownBps != 0 || r.UpBps != 0 {
		t.Errorf("counter reset should clamp rate to 0, got %+v", r)
	}
	if r.TotalRx != 0 || r.TotalTx != 0 {
		t.Errorf("totals after reset should be 0, got %d/%d", r.TotalRx, r.TotalTx)
	}
}

func TestSignalDBM(t *testing.T) {
	cases := map[int]int{0: -100, 100: -50, 50: -75, 94: -53}
	for pct, want := range cases {
		if got := (WiFiInfo{SignalPercent: pct}).SignalDBM(); got != want {
			t.Errorf("SignalDBM(%d%%) = %d, want %d", pct, got, want)
		}
	}
}

func TestSameChannelCount(t *testing.T) {
	w := WiFiInfo{
		Channel: 149,
		Nearby: []AccessPoint{
			{SSID: "a", Channel: 149},
			{SSID: "b", Channel: 149},
			{SSID: "c", Channel: 36},
		},
	}
	if got := w.SameChannelCount(); got != 2 {
		t.Errorf("SameChannelCount = %d, want 2", got)
	}
}

func TestRecommendChannel(t *testing.T) {
	// 2.4 GHz, crowded on 1 and 6, so 11 should win.
	w24 := WiFiInfo{Channel: 1, Nearby: []AccessPoint{
		{Channel: 1}, {Channel: 1}, {Channel: 6}, {Channel: 3}, // 3 overlaps both 1 and 6
	}}
	if got := w24.RecommendChannel(); got != 11 {
		t.Errorf("2.4GHz recommend = %d, want 11", got)
	}

	// 5 GHz, current 149 crowded, 36 in use but quieter -> suggest 36.
	w5 := WiFiInfo{Channel: 149, Nearby: []AccessPoint{
		{Channel: 149}, {Channel: 149}, {Channel: 36},
	}}
	if got := w5.RecommendChannel(); got != 36 {
		t.Errorf("5GHz recommend = %d, want 36", got)
	}

	// Already clearest: keep current.
	wClear := WiFiInfo{Channel: 36, Nearby: []AccessPoint{{Channel: 149}}}
	if got := wClear.RecommendChannel(); got != 36 {
		t.Errorf("clear channel recommend = %d, want 36 (keep)", got)
	}
}
