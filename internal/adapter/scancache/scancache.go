// Package scancache remembers the last scan's IP-to-MAC map so that aliasing a
// device by IP can skip a fresh ARP sweep.
//
// ponytail: best-effort and disposable. A missing or corrupt cache just makes
// the caller fall back to scanning, so there is no atomic write and no locking.
// The map reflects the last scan, so a MAC read back here can be stale if the
// device left or the IP was reassigned since; callers show the resolved MAC so
// a wrong hit is visible.
package scancache

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"

	"github.com/gsjonio/netwp/internal/core"
)

// DefaultPath returns <user-config-dir>/netwp/lastscan.json, creating the
// netwp directory.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "netwp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "lastscan.json"), nil
}

// entry is the JSON-friendly shape of a cached device: net.IP/net.HardwareAddr
// don't round-trip usefully through encoding/json (HardwareAddr would encode
// as base64), so the cache stores strings instead, same reasoning as
// internal/tui/json.go's deviceJSON. Only the fields Diff and Lookup need.
type entry struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Hostname string `json:"hostname,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
}

// Save writes a snapshot of devices to path, replacing any prior cache.
func Save(path string, devices []core.Device) error {
	entries := make([]entry, 0, len(devices))
	for _, d := range devices {
		if len(d.MAC) == 0 {
			continue
		}
		entries = append(entries, entry{IP: d.IP.String(), MAC: d.MAC.String(), Hostname: d.Hostname, Vendor: d.Vendor})
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads back the last snapshot saved by Save, as []core.Device with only
// IP, MAC, Hostname and Vendor populated. Returns nil, err if the cache is
// absent, unreadable, or corrupt.
func Load(path string) ([]core.Device, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	devices := make([]core.Device, 0, len(entries))
	for _, e := range entries {
		mac, err := net.ParseMAC(e.MAC)
		if err != nil {
			continue
		}
		devices = append(devices, core.Device{IP: net.ParseIP(e.IP), MAC: mac, Hostname: e.Hostname, Vendor: e.Vendor})
	}
	return devices, nil
}

// Lookup returns the cached MAC for ip, or false if the cache is absent,
// unreadable, or has no entry for ip.
func Lookup(path string, ip net.IP) (net.HardwareAddr, bool) {
	devices, err := Load(path)
	if err != nil {
		return nil, false
	}
	for _, d := range devices {
		if d.IP.Equal(ip) {
			return d.MAC, true
		}
	}
	return nil, false
}
