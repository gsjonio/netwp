// Package wifi reports the active wireless connection and nearby access points.
//
// The parsing is kept build-tag-free and pure (string in, struct out) so it can
// be unit tested from any OS. The netsh/nmcli calls that produce the strings
// live in the platform files.
package wifi

import (
	"strconv"
	"strings"

	"github.com/gsjonio/netwp/internal/core"
)

// kv splits a "key : value" netsh line on its first colon. The value may itself
// contain colons (a BSSID), so only the first colon is a separator. Keys are
// lowercased so matching is locale-case-insensitive.
func kv(line string) (key, val string, ok bool) {
	k, v, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v), true
}

// firstInt returns the first integer found in s (handles "270", "44 ", "40%").
func firstInt(s string) int {
	start := -1
	for i, r := range s {
		if r >= '0' && r <= '9' {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			n, _ := strconv.Atoi(s[start:i])
			return n
		}
	}
	if start >= 0 {
		n, _ := strconv.Atoi(s[start:])
		return n
	}
	return 0
}

// parseInterfaces reads `netsh wlan show interfaces` output. It tolerates both
// English and Portuguese field labels.
func parseInterfaces(out string) core.WiFiInfo {
	var w core.WiFiInfo
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := kv(line)
		if !ok {
			continue
		}
		switch k {
		case "ssid":
			w.SSID = v
		case "bssid", "ap bssid":
			w.BSSID = v
		case "signal", "sinal":
			w.SignalPercent = firstInt(v)
		case "channel", "canal":
			w.Channel = firstInt(v)
		case "band", "banda":
			w.Band = v
		case "radio type", "tipo de rádio":
			w.RadioType = v
		}
		switch {
		case strings.HasPrefix(k, "receive rate"), strings.HasPrefix(k, "taxa de recepção"), strings.HasPrefix(k, "taxa de recebimento"):
			w.RxRateMbps = firstInt(v)
		case strings.HasPrefix(k, "transmit rate"), strings.HasPrefix(k, "taxa de transmiss"):
			w.TxRateMbps = firstInt(v)
		}
	}
	// A connected interface always lists an SSID; a disconnected one does not.
	w.Connected = w.SSID != ""
	return w
}

// parseNetworks reads `netsh wlan show networks mode=bssid` output into the list
// of visible APs (one entry per BSSID), for interference context.
func parseNetworks(out string) []core.AccessPoint {
	var aps []core.AccessPoint
	curSSID := ""
	curSignal := 0
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := kv(line)
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(k, "ssid ") && !strings.HasPrefix(k, "bssid"):
			// "SSID 1", "SSID 2", ... a new network name.
			curSSID = v
			curSignal = 0
		case k == "signal" || k == "sinal":
			curSignal = firstInt(v)
		case k == "channel" || k == "canal":
			// One channel line per BSSID: emit an AP for it.
			aps = append(aps, core.AccessPoint{SSID: curSSID, Channel: firstInt(v), SignalPercent: curSignal})
		}
	}
	return aps
}
