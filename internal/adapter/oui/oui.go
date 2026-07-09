// Package oui resolves a MAC address to its manufacturer using the IEEE OUI
// registry embedded at build time.
package oui

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/csv"
	"fmt"
	"net"
	"strings"
	"sync"
)

// The IEEE MA-L registry (~35k entries), gzipped. Columns:
// Registry,Assignment,Organization Name,Organization Address.
// Regenerate:
//
//	curl -o data/oui.csv https://standards-oui.ieee.org/oui/oui.csv
//	gzip -9 -c data/oui.csv > data/oui.csv.gz && rm data/oui.csv
//
//go:embed data/oui.csv.gz
var registryGZ []byte

// Lookup implements core.VendorLookup. The registry is decoded lazily on first
// use, so commands that never resolve a vendor pay nothing.
type Lookup struct{}

func New() Lookup { return Lookup{} }

var (
	once  sync.Once
	table map[string]string // OUI prefix (uppercase hex, no separators) -> vendor
)

func load() {
	table = make(map[string]string, 40000)

	gz, err := gzip.NewReader(bytes.NewReader(registryGZ))
	if err != nil {
		return // ponytail: embedded data is trusted; leave table empty on failure
	}
	defer gz.Close() //nolint:errcheck // best-effort cleanup, reading from an in-memory buffer

	r := csv.NewReader(gz)
	r.FieldsPerRecord = -1 // organization address column count varies
	_, _ = r.Read()        // skip header
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		if len(record) < 3 {
			continue
		}
		table[strings.ToUpper(record[1])] = record[2]
	}
}

// Vendor returns the manufacturer for mac, or "Unknown" if the OUI is unlisted.
func (Lookup) Vendor(mac net.HardwareAddr) string {
	if len(mac) < 3 {
		return ""
	}
	once.Do(load)
	prefix := strings.ToUpper(fmt.Sprintf("%02X%02X%02X", mac[0], mac[1], mac[2]))
	if v, ok := table[prefix]; ok {
		return v
	}
	return "Unknown"
}
