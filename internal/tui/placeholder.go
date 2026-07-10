package tui

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

// dash is the placeholder shown for an empty/unknown value in every table.
const dash = "—"

// orDash returns s, or the placeholder when s is empty.
func orDash(s string) string {
	if s == "" {
		return dash
	}
	return s
}

// macText renders a MAC, or the placeholder when it is absent.
func macText(m net.HardwareAddr) string {
	if len(m) == 0 {
		return dash
	}
	return m.String()
}

// classLabel renders a device class, or the placeholder when it is unknown.
func classLabel(c core.DeviceClass) string {
	if c == core.ClassUnknown {
		return dash
	}
	return c.String()
}

// rttText renders a round-trip time. Sub-millisecond LAN replies show as "<1ms";
// an unreachable host shows the placeholder.
func rttText(rtt time.Duration, reachable bool) string {
	if !reachable {
		return dash
	}
	if rtt < time.Millisecond {
		return "<1ms"
	}
	return fmt.Sprintf("%dms", rtt.Milliseconds())
}

// rttQuality buckets round-trip time into a coarse tier for coloring.
type rttQuality int

const (
	rttUnknown rttQuality = iota // unreachable, or RTT not measured
	rttGood
	rttMedium
	rttBad
)

// rttQualityOf classifies rtt for LAN traffic: even "bad" here (>=100ms) is
// fast by internet standards, but on your own network it usually means
// congestion, a weak Wi-Fi link, or a struggling device -- worth a second
// look. Single source of truth for both table renderers' color thresholds.
func rttQualityOf(rtt time.Duration, reachable bool) rttQuality {
	if !reachable {
		return rttUnknown
	}
	switch {
	case rtt < 20*time.Millisecond:
		return rttGood
	case rtt < 100*time.Millisecond:
		return rttMedium
	default:
		return rttBad
	}
}

// portsText renders a device's open ports as a compact comma-separated list
// (e.g. "80,443"), or the placeholder when none were found/probed. Full
// per-port names are one level down, via `netwp ports <ip>`.
func portsText(ports []int) string {
	if len(ports) == 0 {
		return dash
	}
	strs := make([]string, len(ports))
	for i, p := range ports {
		strs[i] = strconv.Itoa(p)
	}
	return strings.Join(strs, ",")
}

// sensitiveTCPPorts flags open ports worth a visual nudge: remote-access and
// file-sharing services (SSH, SMB, RDP) whose exposure on a home network is
// usually unintentional. A display concern, not domain classification, so
// it's kept local to tui even though it overlaps core's classification list.
var sensitiveTCPPorts = map[int]bool{22: true, 445: true, 3389: true}

// hasSensitivePort reports whether ports includes one worth flagging.
func hasSensitivePort(ports []int) bool {
	for _, p := range ports {
		if sensitiveTCPPorts[p] {
			return true
		}
	}
	return false
}
