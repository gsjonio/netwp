// Package classstore persists user-pinned device classes in a JSON file, keyed
// by MAC address, so a manual override (e.g. "this is my phone") survives DHCP
// address changes and outlives the automatic guess.
//
// ponytail: same flat-JSON-map design as aliasstore, and right for the same
// reason: a handful of hand-set overrides, human-editable, loaded once,
// rewritten whole on each change. Revisit only at thousands of entries.
package classstore

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

// Store is a MAC-to-class map backed by a JSON file. Safe for concurrent use
// within one process.
type Store struct {
	path string
	mu   sync.RWMutex
	m    map[string]string // canonical MAC -> class name (as core.DeviceClass.String)
}

// DefaultPath returns <user-config-dir>/netwp/classoverride.json, creating the
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
	return filepath.Join(dir, "classoverride.json"), nil
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

func key(mac net.HardwareAddr) string {
	return strings.ToLower(mac.String())
}

// ClassOverride implements core.ClassLookup.
func (s *Store) ClassOverride(mac net.HardwareAddr) (core.DeviceClass, bool) {
	if len(mac) == 0 {
		return core.ClassUnknown, false
	}
	s.mu.RLock()
	name, ok := s.m[key(mac)]
	s.mu.RUnlock()
	if !ok {
		return core.ClassUnknown, false
	}
	return core.ParseClass(name)
}

// Set pins class to mac and persists the change.
func (s *Store) Set(mac net.HardwareAddr, class core.DeviceClass) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key(mac)] = class.String()
	return s.save()
}

// Delete removes any pinned class for mac and persists the change.
func (s *Store) Delete(mac net.HardwareAddr) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, key(mac))
	return s.save()
}

// List returns all pinned classes sorted by MAC. Entries with an unparseable
// class (a hand-edited typo) are skipped.
func (s *Store) List() []core.ClassPin {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]core.ClassPin, 0, len(s.m))
	for k, name := range s.m {
		mac, err := net.ParseMAC(k)
		if err != nil {
			continue
		}
		class, ok := core.ParseClass(name)
		if !ok {
			continue
		}
		out = append(out, core.ClassPin{MAC: mac, Class: class})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MAC.String() < out[j].MAC.String() })
	return out
}

// save writes the map to disk atomically (temp file + rename) so a crash mid
// write cannot corrupt the existing overrides.
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
