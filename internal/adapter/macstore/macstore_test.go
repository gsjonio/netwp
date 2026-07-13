package macstore

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func mustMAC(t *testing.T, s string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(s)
	if err != nil {
		t.Fatalf("ParseMAC(%q): %v", s, err)
	}
	return mac
}

func TestOpenMissingFileIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "none.json")
	s, err := Open[string](path)
	if err != nil {
		t.Fatalf("Open on a missing file should not error, got %v", err)
	}
	if len(s.Entries()) != 0 {
		t.Errorf("missing file should yield an empty map, got %d entries", len(s.Entries()))
	}
}

func TestSetGetRoundTripAndCanonicalKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	s, _ := Open[string](path)
	if err := s.Set(mustMAC(t, "AA:BB:CC:DD:EE:FF"), "PC"); err != nil {
		t.Fatal(err)
	}

	// A lookup with different casing hits the same canonical key.
	if v, ok := s.Get(mustMAC(t, "aa:bb:cc:dd:ee:ff")); !ok || v != "PC" {
		t.Errorf("Get = (%q, %v), want (PC, true) regardless of case", v, ok)
	}
	// An empty MAC never matches.
	if _, ok := s.Get(nil); ok {
		t.Error("Get(nil) should return false")
	}

	// The value survives a reload from disk.
	reopened, err := Open[string](path)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := reopened.Get(mustMAC(t, "aa:bb:cc:dd:ee:ff")); !ok || v != "PC" {
		t.Errorf("after reload Get = (%q, %v), want (PC, true)", v, ok)
	}
}

func TestDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	s, _ := Open[string](path)
	mac := mustMAC(t, "aa:bb:cc:dd:ee:01")
	_ = s.Set(mac, "x")
	if err := s.Delete(mac); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Get(mac); ok {
		t.Error("value should be gone after Delete")
	}
}

func TestEntriesSortedAndSkipsInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	// Hand-write a file with a bad key mixed in, as a manual edit could.
	if err := os.WriteFile(path, []byte(`{"aa:bb:cc:dd:ee:02":"b","aa:bb:cc:dd:ee:01":"a","garbage":"skip"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open[string](path)
	if err != nil {
		t.Fatal(err)
	}
	entries := s.Entries()
	if len(entries) != 2 {
		t.Fatalf("Entries = %d, want 2 (the invalid key skipped)", len(entries))
	}
	if entries[0].MAC.String() != "aa:bb:cc:dd:ee:01" || entries[1].MAC.String() != "aa:bb:cc:dd:ee:02" {
		t.Errorf("Entries not sorted by MAC: %v, %v", entries[0].MAC, entries[1].MAC)
	}
}
