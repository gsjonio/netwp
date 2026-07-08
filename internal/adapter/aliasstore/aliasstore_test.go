package aliasstore

import (
	"net"
	"path/filepath"
	"testing"
)

func mustMAC(t *testing.T, s string) net.HardwareAddr {
	t.Helper()
	mac, err := net.ParseMAC(s)
	if err != nil {
		t.Fatalf("bad MAC %q: %v", s, err)
	}
	return mac
}

func TestSetGetPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	mac := mustMAC(t, "aa:bb:cc:dd:ee:ff")

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Alias(mac); got != "" {
		t.Errorf("empty store returned %q, want empty", got)
	}
	if err := s.Set(mac, "Living Room TV"); err != nil {
		t.Fatal(err)
	}

	// Reopen to prove it hit disk.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := s2.Alias(mac); got != "Living Room TV" {
		t.Errorf("after reopen: alias = %q, want %q", got, "Living Room TV")
	}
}

func TestLookupIsCaseInsensitive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	s, _ := Open(path)
	if err := s.Set(mustMAC(t, "AA:BB:CC:DD:EE:FF"), "PC"); err != nil {
		t.Fatal(err)
	}
	// Same address, different case/formatting on lookup.
	if got := s.Alias(mustMAC(t, "aa-bb-cc-dd-ee-ff")); got != "PC" {
		t.Errorf("case-insensitive lookup failed: got %q", got)
	}
}

func TestDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	s, _ := Open(path)
	mac := mustMAC(t, "11:22:33:44:55:66")
	s.Set(mac, "Phone")
	if err := s.Delete(mac); err != nil {
		t.Fatal(err)
	}
	if got := s.Alias(mac); got != "" {
		t.Errorf("after delete: alias = %q, want empty", got)
	}
}

func TestList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "aliases.json")
	s, _ := Open(path)
	s.Set(mustMAC(t, "22:22:22:22:22:22"), "B")
	s.Set(mustMAC(t, "11:11:11:11:11:11"), "A")

	list := s.List()
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}
	if list[0].Name != "A" || list[1].Name != "B" {
		t.Errorf("List not sorted by MAC: %+v", list)
	}
}

func TestOpenMissingFile(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("Open of missing file should succeed, got %v", err)
	}
	if got := len(s.List()); got != 0 {
		t.Errorf("missing file should be empty, got %d entries", got)
	}
}
