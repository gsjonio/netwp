package core

import (
	"net"
	"testing"
	"time"
)

func dev(mac, ip string) Device {
	m, _ := net.ParseMAC(mac)
	return Device{IP: net.ParseIP(ip).To4(), MAC: m, Online: true}
}

func TestTrackerLifecycle(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	t0 := time.Unix(0, 0)
	a := dev("aa:aa:aa:aa:aa:aa", "192.168.0.2")
	b := dev("bb:bb:bb:bb:bb:bb", "192.168.0.3")

	// First scan: both are new -> two Joined.
	if ev := tr.Observe([]Device{a, b}, t0); len(ev) != 2 {
		t.Fatalf("first scan: %d events, want 2 (Joined)", len(ev))
	}

	// Same devices again: nothing changed.
	if ev := tr.Observe([]Device{a, b}, t0.Add(10*time.Second)); len(ev) != 0 {
		t.Fatalf("stable scan: %d events, want 0", len(ev))
	}

	// b missing but still inside the grace window (last seen 10s ago < 30s).
	if ev := tr.Observe([]Device{a}, t0.Add(20*time.Second)); len(ev) != 0 {
		t.Fatalf("within grace: %d events, want 0", len(ev))
	}

	// b missing beyond the grace window -> one Left.
	ev := tr.Observe([]Device{a}, t0.Add(60*time.Second))
	if len(ev) != 1 || ev[0].Kind != Left {
		t.Fatalf("beyond grace: got %v, want one Left", ev)
	}

	// b returns -> one Joined.
	ev = tr.Observe([]Device{a, b}, t0.Add(70*time.Second))
	if len(ev) != 1 || ev[0].Kind != Joined {
		t.Fatalf("return: got %v, want one Joined", ev)
	}

	// Both devices tracked, sorted by IP.
	ds := tr.Devices()
	if len(ds) != 2 || !ds[0].IP.Equal(net.ParseIP("192.168.0.2")) {
		t.Fatalf("Devices() = %v, want a before b", ds)
	}
}
