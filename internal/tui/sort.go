package tui

import (
	"bytes"
	"sort"
	"strings"

	"github.com/gsjonio/netwp/internal/core"
)

// sortKey is the column the device table is ordered by. The user cycles it with
// `s` in monitor/dashboard; online devices always sort ahead of offline ones, so
// this only orders within each group.
type sortKey int

const (
	sortIP sortKey = iota
	sortRTT
	sortName
	sortClass
)

func (k sortKey) String() string {
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
func (k sortKey) next() sortKey { return (k + 1) % 4 }

// displayName is the name shown for a device: alias, else hostname, else its IP.
func displayName(d core.TrackedDevice) string {
	if d.Alias != "" {
		return d.Alias
	}
	if d.Hostname != "" {
		return d.Hostname
	}
	return d.IP.String()
}

// sortDevices orders devices in place by key, online first. RTT puts reachable
// devices ahead of unreachable ones and sorts by round-trip ascending (fastest
// first); the others sort ascending on their field. Stable, so equal rows keep
// their prior (IP) order.
func sortDevices(devices []core.TrackedDevice, key sortKey) {
	sort.SliceStable(devices, func(i, j int) bool {
		a, b := devices[i], devices[j]
		if a.Online != b.Online {
			return a.Online // online rows first, regardless of key
		}
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
	})
}
