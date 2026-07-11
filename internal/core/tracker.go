package core

import (
	"bytes"
	"sort"
	"time"
)

// TrackedDevice is a device plus its presence history across scans.
type TrackedDevice struct {
	Device
	FirstSeen time.Time
	LastSeen  time.Time
	Online    bool
}

// EventKind classifies a presence change.
type EventKind int

const (
	Joined EventKind = iota // appeared for the first time, or came back online
	Left                    // went offline after the grace period elapsed
)

// deviceRetention is how long a departed device is remembered before the
// tracker forgets it. Without eviction the device map only grows across a long
// monitor/dashboard session -- unbounded under MAC randomization (phones rotate
// MACs) or a device spoofing many addresses. Much larger than any offlineAfter,
// so a device that just left keeps its "left N ago" history for a good while.
const deviceRetention = 30 * time.Minute

// Event is a presence change emitted by the Tracker.
type Event struct {
	Kind   EventKind
	Device Device
	At     time.Time
}

// Tracker folds successive scans into a stable device set and reports join/leave
// events. Pure logic: the caller supplies the clock, so it is fully testable.
// Not safe for concurrent use — drive it from a single goroutine.
type Tracker struct {
	devices      map[string]*TrackedDevice // keyed by MAC string
	offlineAfter time.Duration
}

// NewTracker returns a Tracker that marks a device offline only once it has been
// missing for at least offlineAfter — a grace window so a single missed scan
// (common on Wi-Fi) does not flap a device in and out.
func NewTracker(offlineAfter time.Duration) *Tracker {
	return &Tracker{devices: map[string]*TrackedDevice{}, offlineAfter: offlineAfter}
}

// Observe folds one scan result into the tracker and returns the events it
// produced (new arrivals, returns, and departures past the grace period).
func (t *Tracker) Observe(scanned []Device, now time.Time) []Event {
	present := make(map[string]bool, len(scanned))
	var events []Event

	for _, d := range scanned {
		key := d.MAC.String()
		present[key] = true

		td, known := t.devices[key]
		if !known {
			t.devices[key] = &TrackedDevice{Device: d, FirstSeen: now, LastSeen: now, Online: true}
			events = append(events, Event{Joined, d, now})
			continue
		}
		td.Device = d // refresh hostname/vendor in case they resolved later
		td.LastSeen = now
		if !td.Online {
			td.Online = true
			events = append(events, Event{Joined, d, now})
		}
	}

	for key, td := range t.devices {
		if present[key] {
			continue // still on the network
		}
		if td.Online {
			if now.Sub(td.LastSeen) >= t.offlineAfter {
				td.Online = false
				events = append(events, Event{Left, td.Device, now})
			}
			continue // online, or still within the grace window
		}
		// Already offline: forget it once it's been gone past the retention
		// window, so the map can't grow without bound in a long session.
		if now.Sub(td.LastSeen) >= deviceRetention {
			delete(t.devices, key)
		}
	}
	return events
}

// Devices returns the tracked devices, online ones first, sorted by IP
// address within each group. On a network with many departed devices,
// interleaving them by IP buries what's actually there among dimmed rows.
func (t *Tracker) Devices() []TrackedDevice {
	out := make([]TrackedDevice, 0, len(t.devices))
	for _, td := range t.devices {
		out = append(out, *td)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Online != out[j].Online {
			return out[i].Online
		}
		return bytes.Compare(out[i].IP.To4(), out[j].IP.To4()) < 0
	})
	return out
}
