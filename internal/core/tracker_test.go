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

// TestTrackerDevicesOnlineFirst checks Devices() groups online devices
// before offline ones (a lower IP that's offline must not sort ahead of a
// higher IP that's online), with IP order preserved within each group.
func TestTrackerDevicesOnlineFirst(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	t0 := time.Unix(0, 0)
	// a has the lowest IP but will go offline; b and c stay online.
	a := dev("aa:aa:aa:aa:aa:aa", "192.168.0.1")
	b := dev("bb:bb:bb:bb:bb:bb", "192.168.0.5")
	c := dev("cc:cc:cc:cc:cc:cc", "192.168.0.9")

	tr.Observe([]Device{a, b, c}, t0)
	tr.Observe([]Device{b, c}, t0.Add(60*time.Second)) // a leaves (past grace)

	ds := tr.Devices()
	if len(ds) != 3 {
		t.Fatalf("Devices() returned %d, want 3", len(ds))
	}
	if !ds[0].Online || !ds[1].Online || ds[2].Online {
		t.Fatalf("Devices() online flags = [%v, %v, %v], want [true, true, false]",
			ds[0].Online, ds[1].Online, ds[2].Online)
	}
	if !ds[0].IP.Equal(net.ParseIP("192.168.0.5")) || !ds[1].IP.Equal(net.ParseIP("192.168.0.9")) {
		t.Errorf("online group order = [%v, %v], want b (.5) before c (.9)", ds[0].IP, ds[1].IP)
	}
	if !ds[2].IP.Equal(net.ParseIP("192.168.0.1")) {
		t.Errorf("offline device = %v, want a (.1) even though it has the lowest IP", ds[2].IP)
	}
}

// TestTrackerEvictsLongGoneDevices checks the device map doesn't grow without
// bound: a device gone past the retention window is forgotten, while one still
// within it is kept (so its recent "left" history survives).
func TestTrackerEvictsLongGoneDevices(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	t0 := time.Unix(0, 0)
	a := dev("aa:aa:aa:aa:aa:aa", "192.168.0.2")

	tr.Observe([]Device{a}, t0)          // joined
	tr.Observe(nil, t0.Add(time.Minute)) // past grace -> offline (Left)
	if len(tr.Devices()) != 1 {          // still remembered within retention
		t.Fatalf("within retention: %d devices, want 1", len(tr.Devices()))
	}
	tr.Observe(nil, t0.Add(deviceRetention+time.Minute)) // past retention -> evicted
	if got := len(tr.Devices()); got != 0 {
		t.Errorf("past retention: %d devices, want 0 (evicted)", got)
	}
}

// TestTrackerEvictionIsBounded feeds many distinct MACs (as MAC randomization
// or spoofing would) and confirms the map stays bounded once they age out,
// instead of growing with every address ever seen. Driven the way a live
// monitor drives it: a scan, then absent scans that first mark them offline
// (past grace) and later evict them (past retention).
func TestTrackerEvictionIsBounded(t *testing.T) {
	tr := NewTracker(30 * time.Second)
	t0 := time.Unix(0, 0)
	var many []Device
	for i := 0; i < 1000; i++ {
		mac := net.HardwareAddr{2, 0, 0, 0, byte(i >> 8), byte(i)}
		many = append(many, Device{IP: net.IPv4(10, 0, byte(i>>8), byte(i)), MAC: mac})
	}
	tr.Observe(many, t0)                 // 1000 join
	tr.Observe(nil, t0.Add(time.Minute)) // absent, past grace -> all offline
	if got := len(tr.Devices()); got != 1000 {
		t.Fatalf("within retention: %d devices, want 1000 still remembered", got)
	}
	tr.Observe(nil, t0.Add(deviceRetention+time.Minute)) // past retention -> all evicted
	if got := len(tr.Devices()); got != 0 {
		t.Errorf("after retention, %d still tracked, want 0 (map must stay bounded)", got)
	}
}
