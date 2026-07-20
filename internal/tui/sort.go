package tui

import (
	"bytes"
	"sort"
	"strings"

	"github.com/gsjonio/netwp/internal/core"
)

// SortKey is the column a device table is ordered by. The live views cycle it
// with `s`; `netwp scan --sort=<column>` picks one up front. Online devices
// always sort ahead of offline ones, so this only orders within each group.
type SortKey int

const (
	sortIP SortKey = iota
	sortRTT
	sortName
	sortClass
)

func (k SortKey) String() string {
	switch k {
	case sortRTT:
		return "RTT"
	case sortName:
		return "name"
	case sortClass:
		return "class"
	default:
		return "IP"
	}
}

// next cycles to the following sort column, wrapping back to IP.
func (k SortKey) next() SortKey { return (k + 1) % 4 }

// ParseSortColumn maps a CLI column name to its key. Reports false for anything
// else, so `--sort=bogus` can fail fast instead of silently sorting by IP.
func ParseSortColumn(s string) (SortKey, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ip":
		return sortIP, true
	case "rtt":
		return sortRTT, true
	case "name":
		return sortName, true
	case "class":
		return sortClass, true
	default:
		return sortIP, false
	}
}

// displayName is the name shown for a device: alias, else hostname, else its IP.
func displayName(d core.Device) string {
	if d.Alias != "" {
		return d.Alias
	}
	if d.Hostname != "" {
		return d.Hostname
	}
	return d.IP.String()
}

// lessByKey compares two devices on one column. RTT puts reachable devices
// ahead of unreachable ones and sorts by round-trip ascending (fastest first);
// the others sort ascending on their field. Shared by both table renderers so
// `--sort=rtt` and the interactive `s` order identically.
func lessByKey(a, b core.Device, key SortKey) bool {
	switch key {
	case sortRTT:
		if a.Reachable != b.Reachable {
			return a.Reachable // reachable (has an RTT) before unreachable
		}
		if !a.Reachable {
			return false // both unreachable: leave order to the stable sort
		}
		return a.RTT < b.RTT
	case sortName:
		return strings.ToLower(displayName(a)) < strings.ToLower(displayName(b))
	case sortClass:
		return a.Class.String() < b.Class.String()
	default: // sortIP
		return bytes.Compare(a.IP.To4(), b.IP.To4()) < 0
	}
}

// sortDevices orders the live views' tracked devices in place, online first.
// Stable, so equal rows keep their prior (IP) order.
func sortDevices(devices []core.TrackedDevice, key SortKey) {
	sort.SliceStable(devices, func(i, j int) bool {
		a, b := devices[i], devices[j]
		if a.Online != b.Online {
			return a.Online // online rows first, regardless of key
		}
		return lessByKey(a.Device, b.Device, key)
	})
}

// SortDevices orders a one-shot scan's devices in place, online first, then by
// key. Exported for `netwp scan --sort`; the renderers print what they are
// given, so the caller decides the order.
func SortDevices(devices []core.Device, key SortKey) {
	sort.SliceStable(devices, func(i, j int) bool {
		a, b := devices[i], devices[j]
		if a.Online != b.Online {
			return a.Online
		}
		return lessByKey(a, b, key)
	})
}
