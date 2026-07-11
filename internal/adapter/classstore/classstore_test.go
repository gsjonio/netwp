package classstore

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/gsjonio/netwp/internal/core"
)

func TestSetLookupDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "classoverride.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	if _, ok := s.ClassOverride(mac); ok {
		t.Fatal("expected no override before Set")
	}
	if err := s.Set(mac, core.ClassMobile); err != nil {
		t.Fatal(err)
	}

	// Reopen to prove it persisted, not just in-memory.
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := s2.ClassOverride(mac); !ok || got != core.ClassMobile {
		t.Errorf("ClassOverride = (%v, %v), want (Mobile, true) after reopen", got, ok)
	}

	if err := s2.Delete(mac); err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.ClassOverride(mac); ok {
		t.Error("expected no override after Delete")
	}
}

func TestListSkipsUnparseable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "classoverride.json")
	s, _ := Open(path)
	mac, _ := net.ParseMAC("11:22:33:44:55:66")
	if err := s.Set(mac, core.ClassPrinter); err != nil {
		t.Fatal(err)
	}
	list := s.List()
	if len(list) != 1 || list[0].Class != core.ClassPrinter {
		t.Errorf("List = %+v, want one Printer pin", list)
	}
}
