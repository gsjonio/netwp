package eventlog

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

func TestFilterByDevice(t *testing.T) {
	entries := []Entry{
		{Kind: "joined", IP: "192.168.1.5", MAC: "aa:bb:cc:dd:ee:01", Name: "Meu PC"},
		{Kind: "left", IP: "192.168.1.6", MAC: "aa:bb:cc:dd:ee:02", Name: "Alexa"},
		{Kind: "joined", IP: "192.168.1.7", MAC: "aa:bb:cc:dd:ee:01", Name: ""}, // same MAC, name not yet set
	}

	// By MAC: matches both entries of that MAC, including the one with no name.
	if got := FilterByDevice(entries, "aa:bb:cc:dd:ee:01", "aa:bb:cc:dd:ee:01"); len(got) != 2 {
		t.Errorf("filter by MAC = %d entries, want 2", len(got))
	}
	// By name (case-insensitive), no resolved MAC: matches on the Name field.
	if got := FilterByDevice(entries, "alexa", ""); len(got) != 1 || got[0].IP != "192.168.1.6" {
		t.Errorf("filter by name = %+v, want the Alexa entry", got)
	}
	// A resolved MAC also catches the entry whose name was empty at log time.
	got := FilterByDevice(entries, "Meu PC", "aa:bb:cc:dd:ee:01")
	if len(got) != 2 {
		t.Errorf("filter by alias-resolved MAC = %d, want 2 (incl. the unnamed one)", len(got))
	}
}

func TestTailAllWhenNonPositive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.jsonl")
	l := New(path)
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	for i := 0; i < 5; i++ {
		if err := l.Log(core.Event{Kind: core.Joined, Device: core.Device{IP: net.IPv4(192, 168, 1, byte(i)), MAC: mac}, At: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	all, err := Tail(path, 0) // 0 => everything
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Errorf("Tail(0) returned %d, want all 5", len(all))
	}
}

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

func TestRotation(t *testing.T) {
	// Shrink the bounds so a handful of events triggers a rotation.
	defer func(l int, b int64) { maxEventLines, rotateAtBytes = l, b }(maxEventLines, rotateAtBytes)
	maxEventLines, rotateAtBytes = 3, 1 // any non-empty file rotates

	path := filepath.Join(t.TempDir(), "events.jsonl")
	l := New(path)
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	for i := 0; i < 10; i++ {
		if err := l.Log(core.Event{Kind: core.Joined, Device: core.Device{IP: net.IPv4(192, 168, 1, byte(i)), MAC: mac}, At: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := Tail(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Rotation keeps the last maxEventLines, then the current Log appends one.
	if len(entries) > maxEventLines+1 {
		t.Errorf("log grew to %d entries, want the file bounded near %d", len(entries), maxEventLines)
	}
	// The most recent event must survive.
	if entries[len(entries)-1].IP != "192.168.1.9" {
		t.Errorf("newest entry = %q, want 192.168.1.9", entries[len(entries)-1].IP)
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
