package scancache

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/gsjonio/netwp/internal/core"
)

func TestSaveAndLookup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lastscan.json")
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	devices := []core.Device{
		{IP: net.IPv4(192, 168, 1, 20), MAC: mac},
		{IP: net.IPv4(192, 168, 1, 21)}, // no MAC: must be skipped
	}
	if err := Save(path, devices); err != nil {
		t.Fatal(err)
	}

	got, ok := Lookup(path, net.IPv4(192, 168, 1, 20))
	if !ok {
		t.Fatal("expected a cache hit for 192.168.1.20")
	}
	if got.String() != mac.String() {
		t.Errorf("cached MAC = %s, want %s", got, mac)
	}

	if _, ok := Lookup(path, net.IPv4(192, 168, 1, 21)); ok {
		t.Error("device without a MAC should not be cached")
	}
	if _, ok := Lookup(path, net.IPv4(10, 0, 0, 1)); ok {
		t.Error("unknown IP should miss")
	}
}

func TestLookupMissingFile(t *testing.T) {
	if _, ok := Lookup(filepath.Join(t.TempDir(), "nope.json"), net.IPv4(1, 1, 1, 1)); ok {
		t.Error("missing cache file should miss, not hit")
	}
}
