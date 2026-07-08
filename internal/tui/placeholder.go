package tui

import (
	"fmt"
	"net"
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
