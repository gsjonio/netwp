// Package eventlog appends device presence-change events (join/leave) to a
// JSONL file, and reads back the most recent ones for `netwp events`.
//
// ponytail: a flat append-only log, not a database and no query engine. It is
// size-bounded (see rotation) so a long-running monitor can't grow it without
// limit, but that's the only concession; richer querying would be a different
// feature.
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
	if err := os.MkdirAll(dir, 0o750); err != nil {
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

// Rotation bounds the log so a long-running monitor can't grow it forever.
// Vars, not consts, so tests can shrink them. Trim by lines but trigger by
// size (a cheap stat, not a full read, on the common path): the file drifts
// between ~maxEventLines and rotateAtBytes, then snaps back.
var (
	maxEventLines       = 5000
	rotateAtBytes int64 = 1 << 20 // ~1 MB
)

// Log appends e to the log file, opening and closing it each call: join/leave
// events are rare enough that this isn't worth a held-open file handle.
func (l Logger) Log(e core.Event) error {
	l.rotateIfLarge()

	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
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

// rotateIfLarge rewrites the log with just its most recent maxEventLines once
// it passes rotateAtBytes. Best-effort: any failure leaves the log untouched,
// so a rotation problem never blocks logging the event itself.
func (l Logger) rotateIfLarge() {
	info, err := os.Stat(l.Path)
	if err != nil || info.Size() < rotateAtBytes {
		return
	}
	lines, err := lastRawLines(l.Path, maxEventLines)
	if err != nil {
		return
	}
	tmp := l.Path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, l.Path) //nolint:errcheck // best-effort; a failed rename leaves the old log in place
}

// lastRawLines returns the last n lines of a file, unparsed, so rotation
// preserves exact bytes instead of re-marshaling (and drops nothing on a line
// that no longer parses).
func lastRawLines(path string, n int) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // best-effort cleanup

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	return lines, scanner.Err()
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
