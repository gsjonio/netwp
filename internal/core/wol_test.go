package core

import (
	"bytes"
	"net"
	"testing"
)

func TestMagicPacket(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	p := MagicPacket(mac)

	if len(p) != 102 { // 6 + 16*6
		t.Fatalf("packet length = %d, want 102", len(p))
	}
	for i := 0; i < 6; i++ {
		if p[i] != 0xFF {
			t.Fatalf("byte %d = %#x, want 0xFF (magic prefix)", i, p[i])
		}
	}
	// Every 6-byte block after the prefix must equal the MAC.
	for off := 6; off < len(p); off += 6 {
		if !bytes.Equal(p[off:off+6], mac) {
			t.Fatalf("block at %d = %x, want %x", off, p[off:off+6], mac)
		}
	}
}

func TestMagicPacketRejectsNon48Bit(t *testing.T) {
	// An 8-byte (EUI-64) or empty address has no WoL magic-packet form.
	if p := MagicPacket(net.HardwareAddr{1, 2, 3, 4, 5, 6, 7, 8}); p != nil {
		t.Errorf("MagicPacket(8 bytes) = %x, want nil", p)
	}
	if p := MagicPacket(nil); p != nil {
		t.Errorf("MagicPacket(nil) = %x, want nil", p)
	}
}
