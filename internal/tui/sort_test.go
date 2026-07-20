package tui

import (
	"net"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

func td(ip byte, online, reachable bool, rtt time.Duration, alias string, class core.DeviceClass) core.TrackedDevice {
	return core.TrackedDevice{
		Device: core.Device{IP: net.IPv4(192, 168, 1, ip), Alias: alias, Class: class, RTT: rtt, Reachable: reachable},
		Online: online,
	}
}

func TestSortDevicesOnlineFirst(t *testing.T) {
	devices := []core.TrackedDevice{
		td(10, false, false, 0, "", core.ClassUnknown),
		td(20, true, true, 5*time.Millisecond, "", core.ClassUnknown),
	}
	sortDevices(devices, sortIP)
	if !devices[0].Online {
		t.Errorf("online device should sort first regardless of key")
	}
}

func TestSortDevicesByRTT(t *testing.T) {
	devices := []core.TrackedDevice{
		td(1, true, true, 30*time.Millisecond, "", core.ClassUnknown),
		td(2, true, false, 0, "", core.ClassUnknown), // unreachable: goes last
		td(3, true, true, 5*time.Millisecond, "", core.ClassUnknown),
	}
	sortDevices(devices, sortRTT)
	if devices[0].RTT != 5*time.Millisecond {
		t.Errorf("fastest device should sort first, got %v", devices[0].RTT)
	}
	if devices[2].Reachable {
		t.Errorf("unreachable device should sort last, got reachable at end")
	}
}

func TestSortDevicesByName(t *testing.T) {
	devices := []core.TrackedDevice{
		td(1, true, true, 0, "Zeta", core.ClassUnknown),
		td(2, true, true, 0, "alpha", core.ClassUnknown),
	}
	sortDevices(devices, sortName)
	if devices[0].Alias != "alpha" {
		t.Errorf("name sort should be case-insensitive ascending, got %q first", devices[0].Alias)
	}
}

func TestParseSortColumn(t *testing.T) {
	for _, in := range []string{"ip", "RTT", " name ", "class"} {
		if _, ok := ParseSortColumn(in); !ok {
			t.Errorf("ParseSortColumn(%q) = not ok, want a valid column", in)
		}
	}
	if _, ok := ParseSortColumn("bogus"); ok {
		t.Error("ParseSortColumn(\"bogus\") = ok, want false so --sort fails fast")
	}
	// An empty flag falls back to IP without being reported as valid, so the
	// caller can tell "not given" from "given but wrong".
	if key, ok := ParseSortColumn(""); ok || key != sortIP {
		t.Errorf("ParseSortColumn(\"\") = (%v, %v), want (IP, false)", key, ok)
	}
}

// TestSortDevicesScanByRTT covers the exported []core.Device path used by
// `scan --sort`, which shares its comparator with the live views.
func TestSortDevicesScanByRTT(t *testing.T) {
	devices := []core.Device{
		{IP: net.IPv4(192, 168, 1, 1), RTT: 30 * time.Millisecond, Reachable: true, Online: true},
		{IP: net.IPv4(192, 168, 1, 2), Reachable: false, Online: true}, // unreachable: last
		{IP: net.IPv4(192, 168, 1, 3), RTT: 5 * time.Millisecond, Reachable: true, Online: true},
	}
	SortDevices(devices, sortRTT)
	if devices[0].RTT != 5*time.Millisecond {
		t.Errorf("fastest device should sort first, got %v", devices[0].RTT)
	}
	if devices[2].Reachable {
		t.Error("unreachable device should sort last")
	}
}

func TestSortDevicesOnlineFirstAndByIP(t *testing.T) {
	devices := []core.Device{
		{IP: net.IPv4(192, 168, 1, 9), Online: false},
		{IP: net.IPv4(192, 168, 1, 5), Online: true},
		{IP: net.IPv4(192, 168, 1, 2), Online: true},
	}
	SortDevices(devices, sortIP)
	if !devices[0].Online || !devices[1].Online || devices[2].Online {
		t.Errorf("online devices should sort first, got %+v", devices)
	}
	if devices[0].IP.String() != "192.168.1.2" {
		t.Errorf("within online, IP order expected; got %v first", devices[0].IP)
	}
}

func TestSortKeyCycles(t *testing.T) {
	k := sortIP
	seen := map[string]bool{}
	for i := 0; i < 4; i++ {
		seen[k.String()] = true
		k = k.next()
	}
	if k != sortIP {
		t.Errorf("next() should cycle back to IP after 4 steps, got %v", k)
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 distinct sort labels, got %d", len(seen))
	}
}
