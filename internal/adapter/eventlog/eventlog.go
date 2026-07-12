// Package eventlog appends device presence-change events (join/leave) to a
// JSONL file, and reads back the most recent ones for `netwp events`.
//
// ponytail: append-only, no rotation, no query engine -- a flat diagnostic
// log, not a database. If this needs bounded size or querying later, that's
// a different feature.
package eventlog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

// DefaultPath returns <user-config-dir>/netwp/events.jsonl, creating the
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
	return filepath.Join(dir, "events.jsonl"), nil
}

// Entry is one recorded presence-change event, the JSON-friendly shape of
// core.Event (net.IP/net.HardwareAddr don't serialize usefully, same
// reasoning as scancache's entry and tui's deviceJSON).
type Entry struct {
	Kind string    `json:"kind"` // "joined" or "left"
	IP   string    `json:"ip"`
	MAC  string    `json:"mac,omitempty"`
	Name string    `json:"name,omitempty"` // alias, else hostname, else empty
	At   time.Time `json:"at"`
}

// Logger implements core.EventLogger by appending JSON lines to Path.
type Logger struct {
	Path string
}

func New(path string) Logger { return Logger{Path: path} }

// Log appends e to the log file, opening and closing it each call: join/leave
// events are rare enough that this isn't worth a held-open file handle.
func (l Logger) Log(e core.Event) error {
	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // best-effort cleanup

	kind := "joined"
	if e.Kind == core.Left {
		kind = "left"
	}
	name := e.Device.Alias
	if name == "" {
		name = e.Device.Hostname
	}
	data, err := json.Marshal(Entry{Kind: kind, IP: e.Device.IP.String(), MAC: e.Device.MAC.String(), Name: name, At: e.At})
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// Tail returns the last n entries in path, oldest first, skipping any
// corrupt lines. n <= 0 returns every entry (used when a caller needs the full
// history to filter it). A missing file returns no entries and no error: no
// events logged yet is not exceptional.
func Tail(path string, n int) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close() //nolint:errcheck // best-effort cleanup

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if n > 0 && len(lines) > n {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(lines))
	for _, line := range lines {
		var e Entry
		if json.Unmarshal([]byte(line), &e) == nil {
			entries = append(entries, e)
		}
	}
	return entries, nil
}

// FilterByDevice keeps only the entries for one device. It matches an entry
// whose MAC equals mac (the caller resolves an alias name to its MAC), or whose
// Name matches device case-insensitively (so it still works for events logged
// before an alias was set, by the name shown at the time). mac may be empty.
func FilterByDevice(entries []Entry, device, mac string) []Entry {
	mac = strings.ToLower(mac)
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if (mac != "" && strings.ToLower(e.MAC) == mac) || strings.EqualFold(e.Name, device) {
			out = append(out, e)
		}
	}
	return out
}
