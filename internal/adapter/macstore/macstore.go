// Package macstore is a MAC-keyed JSON map: the file-backed persistence shared
// by the alias, class-override, and watch stores. It owns the parts those three
// had in triplicate -- the config path, atomic write (temp + rename), canonical
// MAC keys, and concurrency guard -- so each domain store is a thin typed
// wrapper that only adds its own accessor names and value type.
//
// ponytail: a flat JSON map is right for a handful of hand-set entries: zero
// dependencies, human-editable, loaded once, rewritten whole on each change.
// Revisit only at thousands of entries or concurrent writers.
package macstore

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Map is a MAC-to-V map backed by the JSON file at path. Safe for concurrent
// use within one process.
type Map[V any] struct {
	path string
	mu   sync.RWMutex
	m    map[string]V
}

// Entry is one MAC-to-value pair, for iteration.
type Entry[V any] struct {
	MAC   net.HardwareAddr
	Value V
}

// Path returns <user-config-dir>/netwp/<file>, creating the netwp directory.
func Path(file string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "netwp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, file), nil
}

// Open loads the map at path. A missing file yields an empty map; the file is
// created on the first Set.
func Open[V any](path string) (*Map[V], error) {
	s := &Map[V]{path: path, m: map[string]V{}}
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

// Get returns the value stored for mac, and whether one was present.
func (s *Map[V]) Get(mac net.HardwareAddr) (V, bool) {
	if len(mac) == 0 {
		var zero V
		return zero, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.m[key(mac)]
	return v, ok
}

// Set assigns v to mac and persists the change.
func (s *Map[V]) Set(mac net.HardwareAddr, v V) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key(mac)] = v
	return s.save()
}

// Delete removes any value for mac and persists the change.
func (s *Map[V]) Delete(mac net.HardwareAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key(mac))
	return s.save()
}

// Entries returns all pairs sorted by MAC, skipping any key that isn't a valid
// MAC (a hand-edited file could contain one).
func (s *Map[V]) Entries() []Entry[V] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry[V], 0, len(s.m))
	for k, v := range s.m {
		mac, err := net.ParseMAC(k)
		if err != nil {
			continue
		}
		out = append(out, Entry[V]{MAC: mac, Value: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MAC.String() < out[j].MAC.String() })
	return out
}

// save writes the map to disk atomically (temp file + rename) so a crash mid
// write cannot corrupt the existing file. Callers hold s.mu.
func (s *Map[V]) save() error {
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
