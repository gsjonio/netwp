package tui

import (
	"net"
	"testing"

	"github.com/gsjonio/netwp/internal/core"
)

func trackedForFilter() []core.TrackedDevice {
	mac1, _ := net.ParseMAC("aa:bb:cc:dd:ee:01")
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:02")
	return []core.TrackedDevice{
		{Device: core.Device{IP: net.IPv4(192, 168, 1, 5), MAC: mac1, Alias: "Meu PC", Vendor: "GIGA-BYTE", Class: core.ClassComputer}},
		{Device: core.Device{IP: net.IPv4(192, 168, 1, 6), MAC: mac2, Hostname: "chromecast", Vendor: "Google", Class: core.ClassMedia}},
	}
}

func TestFilterDevices(t *testing.T) {
	devs := trackedForFilter()

	cases := map[string]int{
		"":          2, // empty matches all
		"meu":       1, // alias, case-insensitive
		"google":    1, // vendor
		"media":     1, // class
		"192.168.1": 2, // IP substring matches both
		"ee:02":     1, // MAC substring
		"nomatch":   0,
	}
	for q, want := range cases {
		if got := len(filterDevices(devs, q)); got != want {
			t.Errorf("filterDevices(%q) = %d, want %d", q, got, want)
		}
	}
}

func TestApplyFilterKey(t *testing.T) {
	if got := applyFilterKey("me", []rune("u"), false); got != "meu" {
		t.Errorf("append = %q, want meu", got)
	}
	if got := applyFilterKey("meu", nil, true); got != "me" {
		t.Errorf("backspace = %q, want me", got)
	}
	if got := applyFilterKey("", nil, true); got != "" {
		t.Errorf("backspace on empty = %q, want empty", got)
	}
}
