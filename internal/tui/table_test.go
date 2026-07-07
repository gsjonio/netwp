package tui

import (
	"bytes"
	"net"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gsjonio/netwp/internal/core"
)

var ansi = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Every rendered line must have the same visible (ANSI-stripped) rune width,
// regardless of which cells are coloured. This guards the alignment bug where
// colour codes leaked into column-width calculations.
func TestRenderAlignment(t *testing.T) {
	mac, _ := net.ParseMAC("30:56:0f:33:80:cc")
	devices := []core.Device{
		{IP: net.ParseIP("192.168.0.1"), MAC: mac, Hostname: "router.local", Vendor: "TP-Link", Online: true},
		{IP: net.ParseIP("192.168.0.20"), MAC: mac, Hostname: "", Vendor: "", Online: true}, // dashes
	}

	var buf bytes.Buffer
	RenderDevices(&buf, devices)

	var width int
	for i, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		visible := utf8.RuneCountInString(ansi.ReplaceAllString(line, ""))
		if i == 0 {
			width = visible
			continue
		}
		if visible != width {
			t.Errorf("line %d visible width = %d, want %d\n%q", i, visible, width, line)
		}
	}
}
