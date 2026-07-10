package eventlog

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

func TestLogAndTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	l := New(path)
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	at := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)

	if err := l.Log(core.Event{Kind: core.Joined, Device: core.Device{IP: net.IPv4(192, 168, 1, 20), MAC: mac, Hostname: "host.local"}, At: at}); err != nil {
		t.Fatal(err)
	}
	if err := l.Log(core.Event{Kind: core.Left, Device: core.Device{IP: net.IPv4(192, 168, 1, 20), MAC: mac, Hostname: "host.local"}, At: at.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}

	entries, err := Tail(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("Tail returned %d entries, want 2", len(entries))
	}
	if entries[0].Kind != "joined" || entries[0].IP != "192.168.1.20" || entries[0].Name != "host.local" {
		t.Errorf("entries[0] = %+v, want the joined event", entries[0])
	}
	if entries[1].Kind != "left" {
		t.Errorf("entries[1].Kind = %q, want \"left\"", entries[1].Kind)
	}
}

func TestTailLimitsCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	l := New(path)
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	for i := 0; i < 5; i++ {
		if err := l.Log(core.Event{Kind: core.Joined, Device: core.Device{IP: net.IPv4(192, 168, 1, byte(i)), MAC: mac}, At: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := Tail(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("Tail(2) returned %d entries, want 2", len(entries))
	}
	if entries[1].IP != "192.168.1.4" {
		t.Errorf("Tail(2) should keep the newest entries, got %+v", entries)
	}
}

func TestTailMissingFile(t *testing.T) {
	entries, err := Tail(filepath.Join(t.TempDir(), "nope.jsonl"), 10)
	if err != nil {
		t.Errorf("missing file should not be an error, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no entries, got %v", entries)
	}
}
