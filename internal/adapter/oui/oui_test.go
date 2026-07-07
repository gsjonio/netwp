package oui

import (
	"net"
	"testing"
)

func TestVendorLookup(t *testing.T) {
	l := New()

	// A real MA-L assignment (28:6F:B9 = Nokia Shanghai Bell) must resolve to a
	// known vendor — proves the gzip+CSV pipeline populated the table.
	if v := l.Vendor(net.HardwareAddr{0x28, 0x6F, 0xB9, 0x01, 0x02, 0x03}); v == "" || v == "Unknown" {
		t.Errorf("known OUI resolved to %q, want a real vendor", v)
	}

	// Locally-administered / unassigned prefix should be Unknown, not a panic.
	if v := l.Vendor(net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x00}); v != "Unknown" {
		t.Errorf("unassigned OUI = %q, want Unknown", v)
	}

	// Malformed (too short) MAC yields empty, never a slice panic.
	if v := l.Vendor(net.HardwareAddr{0x01}); v != "" {
		t.Errorf("short MAC = %q, want empty", v)
	}

	if len(table) < 20000 {
		t.Errorf("registry has %d entries, expected a full IEEE table", len(table))
	}
}
