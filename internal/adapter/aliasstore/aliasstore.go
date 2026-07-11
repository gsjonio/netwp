// Package aliasstore persists user-defined device nicknames, keyed by MAC so a
// label survives DHCP address changes. It is a thin typed wrapper over
// macstore.Map (which owns the file I/O).
package aliasstore

import (
	"net"

	"github.com/gsjonio/netwp/internal/adapter/macstore"
	"github.com/gsjonio/netwp/internal/core"
)

// Store is a MAC-to-nickname map. Set and Delete are promoted from the embedded
// macstore.Map; Alias and List add the alias-specific shape.
type Store struct {
	*macstore.Map[string]
}

// DefaultPath returns the per-user location of the alias file.
func DefaultPath() (string, error) { return macstore.Path("aliases.json") }

// Open loads the store at path. A missing file yields an empty store.
func Open(path string) (*Store, error) {
	m, err := macstore.Open[string](path)
	if err != nil {
		return nil, err
	}
	return &Store{m}, nil
}

// Alias implements core.AliasLookup: the nickname for mac, or "" if unset.
func (s *Store) Alias(mac net.HardwareAddr) string {
	name, _ := s.Get(mac)
	return name
}

// List returns all stored aliases sorted by MAC.
func (s *Store) List() []core.Alias {
	entries := s.Entries()
	out := make([]core.Alias, len(entries))
	for i, e := range entries {
		out[i] = core.Alias{MAC: e.MAC, Name: e.Value}
	}
	return out
}
