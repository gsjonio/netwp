package main

import (
	"encoding/json"
	"os"

	"github.com/gsjonio/netwp/internal/core"
)

// printJSON writes v as indented JSON to stdout. The read-only commands accept
// --json for scripting; the spinner and any notes go to stderr, so stdout stays
// clean JSON.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// checkJSON is the machine-readable shape of a doctor check.
type checkJSON struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func doctorJSON(checks []core.Check) []checkJSON {
	out := make([]checkJSON, len(checks))
	for i, c := range checks {
		out[i] = checkJSON{Name: c.Name, OK: c.OK, Detail: c.Detail}
	}
	return out
}

// portJSON is one open port with its well-known name.
type portJSON struct {
	Port int    `json:"port"`
	Name string `json:"name"`
}

// portsResultJSON is the machine-readable shape of `netwp ports`.
type portsResultJSON struct {
	IP        string     `json:"ip"`
	Reachable bool       `json:"reachable"`
	RTTMillis *float64   `json:"rtt_ms,omitempty"`
	TTL       int        `json:"ttl,omitempty"`
	Ports     []portJSON `json:"ports"`
}

// speedtestResultJSON is the machine-readable shape of `netwp speedtest`.
type speedtestResultJSON struct {
	DownloadMbps float64 `json:"download_mbps"`
	UploadMbps   float64 `json:"upload_mbps"`
	Edge         string  `json:"edge,omitempty"`
}
