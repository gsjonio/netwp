package tui

import (
	"strings"

	"github.com/gsjonio/netwp/internal/core"
)

// matchesFilter reports whether a device matches the query as a case-insensitive
// substring of any of its visible fields (IP, alias, hostname, vendor, MAC,
// class). An empty query matches everything.
func matchesFilter(d core.TrackedDevice, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	for _, f := range []string{
		d.IP.String(), d.Alias, d.Hostname, d.Vendor, d.MAC.String(), d.Class.String(),
	} {
		if strings.Contains(strings.ToLower(f), q) {
			return true
		}
	}
	return false
}

// filterDevices keeps only the devices matching query (all of them if empty).
func filterDevices(devices []core.TrackedDevice, query string) []core.TrackedDevice {
	if query == "" {
		return devices
	}
	out := make([]core.TrackedDevice, 0, len(devices))
	for _, d := range devices {
		if matchesFilter(d, query) {
			out = append(out, d)
		}
	}
	return out
}

// applyFilterKey folds one keystroke into a filter string while the user is
// typing: printable runes append, backspace deletes the last rune. Esc/Enter
// are handled by the caller (they change modes, not the text).
func applyFilterKey(filter string, runes []rune, backspace bool) string {
	if backspace {
		r := []rune(filter)
		if len(r) > 0 {
			return string(r[:len(r)-1])
		}
		return filter
	}
	return filter + string(runes)
}
