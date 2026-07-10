package tui

import (
	"bytes"
	"net"
	"regexp"
	"strings"
	"testing"
	"time"
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

func TestRttQualityOf(t *testing.T) {
	cases := []struct {
		rtt       time.Duration
		reachable bool
		want      rttQuality
	}{
		{0, false, rttUnknown},
		{0, true, rttGood},
		{19 * time.Millisecond, true, rttGood},
		{20 * time.Millisecond, true, rttMedium},
		{99 * time.Millisecond, true, rttMedium},
		{100 * time.Millisecond, true, rttBad},
		{500 * time.Millisecond, true, rttBad},
	}
	for _, c := range cases {
		if got := rttQualityOf(c.rtt, c.reachable); got != c.want {
			t.Errorf("rttQualityOf(%v, %v) = %v, want %v", c.rtt, c.reachable, got, c.want)
		}
	}
}

func TestRttCellColors(t *testing.T) {
	if got := rttCell(5*time.Millisecond, true); got.color != colorGreen {
		t.Errorf("rttCell(5ms).color = %q, want colorGreen", got.color)
	}
	if got := rttCell(50*time.Millisecond, true); got.color != colorBold {
		t.Errorf("rttCell(50ms).color = %q, want colorBold", got.color)
	}
	if got := rttCell(200*time.Millisecond, true); got.color != colorWarn {
		t.Errorf("rttCell(200ms).color = %q, want colorWarn", got.color)
	}
	if got := rttCell(0, false); got.color != colorDim || got.text != dash {
		t.Errorf("rttCell(unreachable) = %+v, want dimmed placeholder", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 22); got != "short" {
		t.Errorf("truncate(short) = %q, want unchanged", got)
	}
	long := "CLOUD NETWORK TECHNOLOGY SINGAPORE PTE. LTD."
	got := truncate(long, 22)
	if r := []rune(got); len(r) != 22 {
		t.Errorf("truncate(long, 22) has %d runes, want 22: %q", len(r), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncate(long, 22) = %q, want it to end in an ellipsis", got)
	}
}

func TestVendorTextDash(t *testing.T) {
	if got := vendorText(""); got != dash {
		t.Errorf("vendorText(\"\") = %q, want %q", got, dash)
	}
}

func TestTtlHint(t *testing.T) {
	cases := []struct {
		ttl  int
		want string
	}{
		{0, ""},
		{64, "Linux"},
		{60, "Linux"},
		{128, "Windows"},
		{100, "Windows"},
		{255, "network gear"},
		{200, "network gear"},
	}
	for _, c := range cases {
		if got := ttlHint(c.ttl); got != c.want {
			t.Errorf("ttlHint(%d) = %q, want %q", c.ttl, got, c.want)
		}
	}
}

func TestTTLText(t *testing.T) {
	if got := TTLText(0); got != dash {
		t.Errorf("TTLText(0) = %q, want %q", got, dash)
	}
	if got := TTLText(64); got != "64 (Linux)" {
		t.Errorf("TTLText(64) = %q, want \"64 (Linux)\"", got)
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
