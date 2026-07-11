// Package watchstore persists a set of "watched" device MACs in a JSON file:
// devices the user wants to be alerted about when they leave the network (a
// security camera, a server -- something whose absence matters).
//
// ponytail: same flat-JSON design as aliasstore/classstore, a set instead of a
// map-to-value. Rewritten whole on each change; fine for a handful of pins.
package watchstore

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Store is a set of watched MACs backed by a JSON file. Safe for concurrent
// use within one process.
type Store struct {
	path string
	mu   sync.RWMutex
	m    map[string]bool // canonical MAC -> watched
}

// DefaultPath returns <user-config-dir>/netwp/watchlist.json, creating the
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
	return filepath.Join(dir, "watchlist.json"), nil
}

// Open loads the store at path. A missing file yields an empty store.
func Open(path string) (*Store, error) {
	s := &Store{path: path, m: map[string]bool{}}
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

func key(mac net.HardwareAddr) string {
	return strings.ToLower(mac.String())
}

// IsWatched implements core.Watchlist.
func (s *Store) IsWatched(mac net.HardwareAddr) bool {
	if len(mac) == 0 {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.m[key(mac)]
}

// Add marks mac as watched and persists the change.
func (s *Store) Add(mac net.HardwareAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key(mac)] = true
	return s.save()
}

// Remove unwatches mac and persists the change.
func (s *Store) Remove(mac net.HardwareAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key(mac))
	return s.save()
}

// List returns all watched MACs sorted.
func (s *Store) List() []net.HardwareAddr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]net.HardwareAddr, 0, len(s.m))
	for k := range s.m {
		if mac, err := net.ParseMAC(k); err == nil {
			out = append(out, mac)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

// save writes the set to disk atomically (temp file + rename).
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
