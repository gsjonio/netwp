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
		{IP: net.ParseIP("192.168.0.1"), MAC: mac, Hostname: "router.local", Vendor: "TP-Link", Online: true, Ports: []int{80, 443}},
		{IP: net.ParseIP("192.168.0.20"), MAC: mac, Hostname: "", Vendor: "", Online: true}, // dashes, no ports
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

	out := ansi.ReplaceAllString(buf.String(), "")
	if !strings.Contains(out, "80,443") {
		t.Errorf("expected the PORTS column to show \"80,443\", got:\n%s", out)
	}
}

// TestPortsCellColorsSensitivePorts checks portsCell (the raw-ANSI table.go
// renderer, unlike lipgloss-based portsCellText) actually applies colorWarn
// when a sensitive port is present, and leaves an ordinary port list plain.
func TestPortsCellColorsSensitivePorts(t *testing.T) {
	if got := portsCell([]int{80, 443}); got.color != "" {
		t.Errorf("portsCell([80,443]).color = %q, want no color", got.color)
	}
	if got := portsCell([]int{22, 80}); got.color != colorWarn {
		t.Errorf("portsCell([22,80]).color = %q, want colorWarn (sensitive port present)", got.color)
	}
	if got := portsCell(nil); got.color != colorDim {
		t.Errorf("portsCell(nil).color = %q, want colorDim (placeholder dash)", got.color)
	}
}

func TestPortsText(t *testing.T) {
	if got := portsText(nil); got != dash {
		t.Errorf("portsText(nil) = %q, want %q", got, dash)
	}
	if got := portsText([]int{22}); got != "22" {
		t.Errorf("portsText([22]) = %q, want \"22\"", got)
	}
	if got := portsText([]int{80, 443, 8009}); got != "80,443,8009" {
		t.Errorf("portsText([80,443,8009]) = %q, want \"80,443,8009\"", got)
	}
}
