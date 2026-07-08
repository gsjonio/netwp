// Package aliasstore persists user-defined device nicknames in a JSON file,
// keyed by MAC address so a label survives DHCP address changes.
//
// ponytail: a flat JSON map is right for a handful of hand-set nicknames:
// zero dependencies, human-editable, loaded once. Ceilings, none of which
// bite at this scale: the whole file is rewritten on every Set/Delete, and
// there is no cross-process lock. Move to an embedded KV or SQLite only if
// this ever grows to thousands of entries or concurrent writers.
package aliasstore

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/gsjonio/netwp/internal/core"
)

// Store is a MAC-to-nickname map backed by a JSON file. Safe for concurrent
// use within one process.
type Store struct {
	path string
	mu   sync.RWMutex
	m    map[string]string // canonical MAC -> name
}

// DefaultPath returns the per-user location of the alias file
// (<user-config-dir>/netwp/aliases.json), creating the netwp directory.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "netwp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "aliases.json"), nil
}

// Open loads the store at path. A missing file yields an empty store; the file
// is created on the first Set.
func Open(path string) (*Store, error) {
	s := &Store{path: path, m: map[string]string{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s.m); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// key normalizes a MAC to its canonical lowercase colon form so lookups match
// regardless of how the address was formatted on input.
func key(mac net.HardwareAddr) string {
	return strings.ToLower(mac.String())
}

// Alias implements core.AliasLookup.
func (s *Store) Alias(mac net.HardwareAddr) string {
	if len(mac) == 0 {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[key(mac)]
}

// Set assigns name to mac and persists the change.
func (s *Store) Set(mac net.HardwareAddr, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key(mac)] = name
	return s.save()
}

// Delete removes any nickname for mac and persists the change.
func (s *Store) Delete(mac net.HardwareAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key(mac))
	return s.save()
}

// List returns all stored aliases sorted by MAC.
func (s *Store) List() []core.Alias {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.Alias, 0, len(s.m))
	for k, name := range s.m {
		mac, err := net.ParseMAC(k)
		if err != nil {
			continue
		}
		out = append(out, core.Alias{MAC: mac, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MAC.String() < out[j].MAC.String() })
	return out
}

// save writes the map to disk atomically (temp file + rename) so a crash mid
// write cannot corrupt the existing aliases.
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.m, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
