package tui

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/gsjonio/netwp/internal/core"
)

// deviceCSVHeader names the columns of `netwp scan --csv`, in order. Same data
// the table shows, in the shape a spreadsheet wants: an absent value is an
// empty cell, not the table's placeholder glyph, which would sort as text in
// the numeric columns.
var deviceCSVHeader = []string{"ip", "mac", "alias", "hostname", "vendor", "class", "rtt_ms", "ttl", "ports"}

// RenderDevicesCSV writes devices to w as CSV: a header row, then one row per
// device in the order given (the caller orders them; see SortDevices).
func RenderDevicesCSV(w io.Writer, devices []core.Device) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(deviceCSVHeader); err != nil {
		return err
	}
	for _, d := range devices {
		if err := cw.Write(deviceCSVRow(d)); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// deviceCSVRow flattens one device into cells matching deviceCSVHeader. RTT is
// milliseconds, the same unit `scan --json` reports, so the two exports agree.
func deviceCSVRow(d core.Device) []string {
	rtt := ""
	if d.Reachable {
		rtt = strconv.FormatFloat(float64(d.RTT.Microseconds())/1000, 'f', -1, 64)
	}
	ttl := ""
	if d.TTL != 0 {
		ttl = strconv.Itoa(d.TTL)
	}
	return []string{
		d.IP.String(),
		d.MAC.String(),
		d.Alias,
		d.Hostname,
		d.Vendor,
		d.Class.String(),
		rtt,
		ttl,
		portsList(d.Ports),
	}
}
