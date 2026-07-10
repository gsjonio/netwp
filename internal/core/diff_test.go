package core

import (
	"net"
	"testing"
)

func mustMAC(t *testing.T, s string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(s)
	if err != nil {
		t.Fatal(err)
	}
	return mac
}

func TestDiffJoinedAndLeft(t *testing.T) {
	a := mustMAC(t, "aa:aa:aa:aa:aa:aa")
	b := mustMAC(t, "bb:bb:bb:bb:bb:bb")

	previous := []Device{{IP: net.IPv4(192, 168, 1, 10), MAC: a}}
	current := []Device{{IP: net.IPv4(192, 168, 1, 20), MAC: b}}

	d := Diff(previous, current)
	if len(d.Joined) != 1 || d.Joined[0].MAC.String() != b.String() {
		t.Errorf("Joined = %+v, want the new device", d.Joined)
	}
	if len(d.Left) != 1 || d.Left[0].MAC.String() != a.String() {
		t.Errorf("Left = %+v, want the departed device", d.Left)
	}
	if len(d.IPChanged) != 0 || len(d.MACChanged) != 0 || len(d.DupMAC) != 0 {
		t.Errorf("unexpected extra changes: %+v", d)
	}
}

func TestDiffIPChanged(t *testing.T) {
	a := mustMAC(t, "aa:aa:aa:aa:aa:aa")
	previous := []Device{{IP: net.IPv4(192, 168, 1, 10), MAC: a}}
	current := []Device{{IP: net.IPv4(192, 168, 1, 99), MAC: a}} // DHCP re-lease

	d := Diff(previous, current)
	if len(d.IPChanged) != 1 || !d.IPChanged[0].IP.Equal(net.IPv4(192, 168, 1, 99)) {
		t.Errorf("IPChanged = %+v, want the device at its new IP", d.IPChanged)
	}
	if len(d.Joined) != 0 || len(d.Left) != 0 || len(d.MACChanged) != 0 {
		t.Errorf("a MAC re-leased to a new IP should not also show as joined/left/MACChanged: %+v", d)
	}
}

func TestDiffMACChanged(t *testing.T) {
	a := mustMAC(t, "aa:aa:aa:aa:aa:aa")
	b := mustMAC(t, "bb:bb:bb:bb:bb:bb")
	previous := []Device{{IP: net.IPv4(192, 168, 1, 10), MAC: a}}
	current := []Device{{IP: net.IPv4(192, 168, 1, 10), MAC: b}} // same IP, new MAC

	d := Diff(previous, current)
	if len(d.MACChanged) != 1 || d.MACChanged[0].MAC.String() != b.String() {
		t.Errorf("MACChanged = %+v, want the device now answering at that IP", d.MACChanged)
	}
}

func TestDiffDupMAC(t *testing.T) {
	a := mustMAC(t, "aa:aa:aa:aa:aa:aa")
	current := []Device{
		{IP: net.IPv4(192, 168, 1, 10), MAC: a},
		{IP: net.IPv4(192, 168, 1, 11), MAC: a},
	}

	d := Diff(nil, current)
	if len(d.DupMAC) != 2 {
		t.Errorf("DupMAC = %+v, want both entries sharing MAC %s", d.DupMAC, a)
	}
}

func TestDiffNoChanges(t *testing.T) {
	a := mustMAC(t, "aa:aa:aa:aa:aa:aa")
	devices := []Device{{IP: net.IPv4(192, 168, 1, 10), MAC: a}}

	d := Diff(devices, devices)
	if len(d.Joined)+len(d.Left)+len(d.IPChanged)+len(d.MACChanged)+len(d.DupMAC) != 0 {
		t.Errorf("identical snapshots should produce no changes, got %+v", d)
	}
}

func TestDiffIgnoresDevicesWithoutMAC(t *testing.T) {
	noMAC := []Device{{IP: net.IPv4(192, 168, 1, 10)}}
	d := Diff(noMAC, noMAC)
	if len(d.Joined)+len(d.Left) != 0 {
		t.Errorf("devices with no MAC have no stable identity to diff: got %+v", d)
	}
}
