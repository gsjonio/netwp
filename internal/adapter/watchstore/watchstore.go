// Package watchstore persists the set of "watched" device MACs: devices the
// user wants to be alerted about when they leave. A thin typed wrapper over
// macstore.Map, using bool as a set membership marker.
package watchstore

import (
	"net"

	"github.com/gsjonio/netwp/internal/adapter/macstore"
)

// Store is a set of watched MACs.
type Store struct {
	*macstore.Map[bool]
}

// DefaultPath returns the per-user location of the watch-list file.
func DefaultPath() (string, error) { return macstore.Path("watchlist.json") }

// Open loads the store at path. A missing file yields an empty store.
func Open(path string) (*Store, error) {
	m, err := macstore.Open[bool](path)
	if err != nil {
		return nil, err
	}
	return &Store{m}, nil
}

// IsWatched implements core.Watchlist.
func (s *Store) IsWatched(mac net.HardwareAddr) bool {
	watched, _ := s.Get(mac)
	return watched
}

// Add marks mac as watched.
func (s *Store) Add(mac net.HardwareAddr) error { return s.Set(mac, true) }

// Remove unwatches mac (Delete is promoted, but the domain name reads better).
func (s *Store) Remove(mac net.HardwareAddr) error { return s.Delete(mac) }

// List returns all watched MACs sorted.
func (s *Store) List() []net.HardwareAddr {
	entries := s.Entries()
	out := make([]net.HardwareAddr, len(entries))
	for i, e := range entries {
		out[i] = e.MAC
	}
	return out
}
