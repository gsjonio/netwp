package tui

import (
	"encoding/json"
	"io"

	"github.com/gsjonio/netwp/internal/core"
)

// deviceJSON is the machine-readable shape of core.Device for `netwp scan
// --json`. It exists because the raw types don't serialize usefully:
// net.HardwareAddr has no MarshalText (would encode as base64), and
// time.Duration would encode as raw nanoseconds instead of a readable number.
type deviceJSON struct {
	IP        string   `json:"ip"`
	MAC       string   `json:"mac"`
	Alias     string   `json:"alias,omitempty"`
	Hostname  string   `json:"hostname,omitempty"`
	Vendor    string   `json:"vendor,omitempty"`
	Class     string   `json:"class"`
	RTTMillis *float64 `json:"rtt_ms,omitempty"`
	TTL       int      `json:"ttl,omitempty"`
	Reachable bool     `json:"reachable"`
	Online    bool     `json:"online"`
	Ports     []int    `json:"ports,omitempty"`
}

// RenderDevicesJSON writes devices as an indented JSON array to w.
func RenderDevicesJSON(w io.Writer, devices []core.Device) error {
	out := make([]deviceJSON, len(devices))
	for i, d := range devices {
		dj := deviceJSON{
			IP:        d.IP.String(),
			MAC:       d.MAC.String(),
			Alias:     d.Alias,
			Hostname:  d.Hostname,
			Vendor:    d.Vendor,
			Class:     d.Class.String(),
			TTL:       d.TTL,
			Reachable: d.Reachable,
			Online:    d.Online,
			Ports:     d.Ports,
		}
		if d.Reachable {
			ms := float64(d.RTT.Microseconds()) / 1000
			dj.RTTMillis = &ms
		}
		out[i] = dj
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
