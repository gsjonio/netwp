package tui

import (
	"bytes"
	"encoding/csv"
	"net"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

func TestRenderDevicesCSV(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	devices := []core.Device{
		{
			IP: net.IPv4(192, 168, 1, 5), MAC: mac, Alias: "TV", Hostname: "tv.local",
			Vendor: "Samsung", Class: core.ClassMedia, RTT: 12500 * time.Microsecond,
			TTL: 64, Reachable: true, Online: true, Ports: []int{80, 443},
		},
		{
			IP: net.IPv4(192, 168, 1, 6), MAC: net.HardwareAddr{1, 2, 3, 4, 5, 6},
			Class: core.ClassUnknown, Online: true,
		},
	}

	var buf bytes.Buffer
	if err := RenderDevicesCSV(&buf, devices); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("output is not valid CSV: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3 (header + 2 devices)", len(rows))
	}

	want := []string{"ip", "mac", "alias", "hostname", "vendor", "class", "rtt_ms", "ttl", "ports"}
	for i, h := range want {
		if rows[0][i] != h {
			t.Errorf("header[%d] = %q, want %q", i, rows[0][i], h)
		}
	}

	// A reachable device carries every column, with RTT in the same unit
	// --json reports and the ports as one comma-separated cell.
	got := rows[1]
	for i, cell := range []string{"192.168.1.5", "aa:bb:cc:dd:ee:ff", "TV", "tv.local", "Samsung", "Media", "12.5", "64", "80,443"} {
		if got[i] != cell {
			t.Errorf("row[%d] = %q, want %q", i, got[i], cell)
		}
	}

	// An unreachable, unprobed device leaves RTT/TTL/ports empty rather than
	// writing the table's placeholder glyph into a numeric column.
	got = rows[2]
	for _, i := range []int{6, 7, 8} {
		if got[i] != "" {
			t.Errorf("row[%d] = %q, want an empty cell", i, got[i])
		}
	}
}
