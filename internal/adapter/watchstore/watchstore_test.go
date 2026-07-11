package watchstore

import (
	"net"
	"path/filepath"
	"testing"
)

func TestAddIsWatchedRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watchlist.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	if s.IsWatched(mac) {
		t.Fatal("should not be watched before Add")
	}
	if err := s.Add(mac); err != nil {
		t.Fatal(err)
	}

	// Reopen to prove persistence.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.IsWatched(mac) {
		t.Error("should be watched after Add + reopen")
	}
	if got := s2.List(); len(got) != 1 || got[0].String() != mac.String() {
		t.Errorf("List = %v, want [%s]", got, mac)
	}

	if err := s2.Remove(mac); err != nil {
		t.Fatal(err)
	}
	if s2.IsWatched(mac) {
		t.Error("should not be watched after Remove")
	}
}

func TestIsWatchedEmptyMAC(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "watchlist.json"))
	if s.IsWatched(nil) {
		t.Error("nil MAC should never be watched")
	}
}
