// Package tui renders discovery results as a legible terminal table.
//
// ponytail: manual rune-width padding + a few ANSI codes give aligned, coloured
// output with zero dependencies. text/tabwriter can't do this: it counts the
// bytes of ANSI colour codes toward column width, which breaks alignment. When
// the interactive monitor (fase 2) lands, move to charmbracelet/lipgloss +
// bubbletea for live-updating, styled tables.
package tui

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gsjonio/netwp/internal/core"
)

const (
	colorReset = "\x1b[0m"
	colorGreen = "\x1b[32m"
	colorCyan  = "\x1b[36m"
	colorDim   = "\x1b[90m"
	colorBold  = "\x1b[1m"
	colorWarn  = "\x1b[31m"
)

const columnGap = "  "

// cell is one table cell: its plain text plus an optional ANSI colour. Width is
// always measured on text, so colour never affects alignment.
//
// ponytail: assumes one terminal column per rune. Fine for IPs, MACs and ASCII
// hostnames; wide CJK/emoji would need a display-width lib.
type cell struct {
	text  string
	color string // "" means no colour
}

// colorEnabled reports whether the plain table should emit ANSI color. It
// honors the NO_COLOR convention (https://no-color.org): if NO_COLOR is set to
// any value, the table renders without escapes. The live monitor/dashboard use
// lipgloss, which honors NO_COLOR on its own.
func colorEnabled() bool {
	_, noColor := os.LookupEnv("NO_COLOR")
	return !noColor
}

// RenderDevices writes a table of devices to w, in the order given. The caller
// orders them (see SortDevices), so `scan --sort` and the default IP order go
// through the same path.
func RenderDevices(w io.Writer, devices []core.Device) {
	header := []cell{
		{"STATUS", colorBold}, {"IP", colorBold}, {"ALIAS", colorBold}, {"RTT", colorBold}, {"TTL", colorBold},
		{"MAC", colorBold}, {"CLASS", colorBold}, {"HOSTNAME", colorBold}, {"VENDOR", colorBold}, {"PORTS", colorBold},
	}
	rows := make([][]cell, 0, len(devices))
	for _, d := range devices {
		rows = append(rows, []cell{
			statusCell(d.Online),
			{d.IP.String(), ""},
			aliasCell(d.Alias),
			rttCell(d.RTT, d.Reachable),
			dashCell(TTLText(d.TTL)),
			macCell(d.MAC),
			classCell(d.Class),
			textCell(d.Hostname),
			dashCell(vendorText(d.Vendor)),
			portsCell(d.Ports),
		})
	}

	widths := columnWidths(header, rows)
	colorize := colorEnabled()
	writeRow(w, header, widths, colorize)
	for _, row := range rows {
		writeRow(w, row, widths, colorize)
	}
}

// columnWidths returns the max rune width of each column across header and rows.
func columnWidths(header []cell, rows [][]cell) []int {
	widths := make([]int, len(header))
	for _, row := range append([][]cell{header}, rows...) {
		for i, c := range row {
			if n := utf8.RuneCountInString(c.text); n > widths[i] {
				widths[i] = n
			}
		}
	}
	return widths
}

func writeRow(w io.Writer, cells []cell, widths []int, colorize bool) {
	parts := make([]string, len(cells))
	for i, c := range cells {
		padded := c.text + strings.Repeat(" ", widths[i]-utf8.RuneCountInString(c.text))
		if colorize && c.color != "" {
			padded = c.color + padded + colorReset
		}
		parts[i] = padded
	}
	fmt.Fprintln(w, strings.Join(parts, columnGap))
}

func statusCell(online bool) cell {
	if online {
		return cell{"● online", colorGreen}
	}
	return cell{"○ offline", colorDim}
}

func macCell(m net.HardwareAddr) cell   { return dashCell(macText(m)) }
func textCell(s string) cell            { return dashCell(orDash(s)) }
func classCell(c core.DeviceClass) cell { return dashCell(classLabel(c)) }

// rttCell colors round-trip time by quality tier: green under 20ms, bold
// neutral under 100ms, red beyond that -- otherwise every RTT reads with the
// same visual weight and a struggling device doesn't stand out.
func rttCell(rtt time.Duration, reachable bool) cell {
	text := rttText(rtt, reachable)
	switch rttQualityOf(rtt, reachable) {
	case rttGood:
		return cell{text, colorGreen}
	case rttMedium:
		return cell{text, colorBold}
	case rttBad:
		return cell{text, colorWarn}
	default:
		return cell{dash, colorDim}
	}
}

// portsCell highlights the ports list when it includes a sensitive one (SSH,
// SMB, RDP), so an unintentionally exposed service catches the eye instead of
// blending in with the rest of the row.
func portsCell(ports []int) cell {
	text := portsText(ports)
	if text == dash {
		return cell{dash, colorDim}
	}
	if hasSensitivePort(ports) {
		return cell{text, colorWarn}
	}
	return cell{text, ""}
}

// dashCell dims the placeholder glyph while leaving real values uncoloured, so
// the "—" for a missing value reads as absent rather than as data.
func dashCell(text string) cell {
	if text == dash {
		return cell{dash, colorDim}
	}
	return cell{text, ""}
}

// aliasCell highlights a user-set nickname so it stands out from resolved data.
func aliasCell(alias string) cell {
	if alias == "" {
		return cell{dash, colorDim}
	}
	return cell{alias, colorCyan}
}
