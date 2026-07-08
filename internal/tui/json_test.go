package tui

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

func TestRenderDevicesJSON(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	devices := []core.Device{
		{
			IP: net.IPv4(192, 168, 1, 5), MAC: mac, Alias: "TV",
			Class: core.ClassMedia, RTT: 12500 * time.Microsecond, Reachable: true, Online: true,
		},
		{
			IP: net.IPv4(192, 168, 1, 6), MAC: net.HardwareAddr{1, 2, 3, 4, 5, 6},
			Class: core.ClassUnknown, Reachable: false, Online: true,
		},
	}

	var buf bytes.Buffer
	if err := RenderDevicesJSON(&buf, devices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}

	first := got[0]
	if first["mac"] != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("mac = %v, want colon-separated string", first["mac"])
	}
	if first["alias"] != "TV" {
		t.Errorf("alias = %v", first["alias"])
	}
	if rtt, ok := first["rtt_ms"].(float64); !ok || rtt != 12.5 {
		t.Errorf("rtt_ms = %v, want 12.5", first["rtt_ms"])
	}

	second := got[1]
	if _, present := second["rtt_ms"]; present {
		t.Errorf("unreachable device should omit rtt_ms, got %v", second["rtt_ms"])
	}
	if _, present := second["alias"]; present {
		t.Errorf("empty alias should be omitted, got %v", second["alias"])
	}
}
