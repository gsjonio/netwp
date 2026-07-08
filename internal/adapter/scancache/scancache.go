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

// Save writes the IP-to-MAC map of devices to path, replacing any prior cache.
func Save(path string, devices []core.Device) error {
	m := make(map[string]string, len(devices))
	for _, d := range devices {
		if len(d.MAC) > 0 {
			m[d.IP.String()] = d.MAC.String()
		}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Lookup returns the cached MAC for ip, or false if the cache is absent,
// unreadable, or has no entry for ip.
func Lookup(path string, ip net.IP) (net.HardwareAddr, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var m map[string]string
	if json.Unmarshal(data, &m) != nil {
		return nil, false
	}
	mac, ok := m[ip.String()]
	if !ok {
		return nil, false
	}
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return nil, false
	}
	return hw, true
}
