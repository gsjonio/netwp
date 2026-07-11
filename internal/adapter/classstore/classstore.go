// Package classstore persists user-pinned device classes, keyed by MAC so a
// manual override ("this is my phone") survives DHCP changes and outlives the
// automatic guess. A thin typed wrapper over macstore.Map.
package classstore

import (
	"net"

	"github.com/gsjonio/netwp/internal/adapter/macstore"
	"github.com/gsjonio/netwp/internal/core"
)

// Store is a MAC-to-class map. The class is stored as its String() form so the
// JSON file is human-readable. Delete is promoted from the embedded map.
type Store struct {
	*macstore.Map[string]
}

// DefaultPath returns the per-user location of the class-override file.
func DefaultPath() (string, error) { return macstore.Path("classoverride.json") }

// Open loads the store at path. A missing file yields an empty store.
func Open(path string) (*Store, error) {
	m, err := macstore.Open[string](path)
	if err != nil {
		return nil, err
	}
	return &Store{m}, nil
}

// ClassOverride implements core.ClassLookup.
func (s *Store) ClassOverride(mac net.HardwareAddr) (core.DeviceClass, bool) {
	name, ok := s.Get(mac)
	if !ok {
		return core.ClassUnknown, false
	}
	return core.ParseClass(name)
}

// Set pins class to mac (shadows the embedded Set, which takes a string).
func (s *Store) Set(mac net.HardwareAddr, class core.DeviceClass) error {
	return s.Map.Set(mac, class.String())
}

// List returns all pinned classes sorted by MAC, skipping any unparseable
// class (a hand-edited typo).
func (s *Store) List() []core.ClassPin {
	out := make([]core.ClassPin, 0)
	for _, e := range s.Entries() {
		class, ok := core.ParseClass(e.Value)
		if !ok {
			continue
		}
		out = append(out, core.ClassPin{MAC: e.MAC, Class: class})
	}
	return out
}
