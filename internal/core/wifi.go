package core

// WiFiInfo describes the active wireless connection and its radio environment.
// Fields are best-effort: anything the platform does not report stays zero/empty.
type WiFiInfo struct {
	Connected     bool
	SSID          string
	BSSID         string
	Band          string // e.g. "5 GHz"
	RadioType     string // e.g. "802.11ax"
	Channel       int
	SignalPercent int // 0..100 as reported by the OS
	RxRateMbps    int
	TxRateMbps    int
	Nearby        []AccessPoint // other visible APs, for interference context
}

// SignalDBM converts the OS signal percentage to an approximate dBm value using
// the common linear mapping Windows itself uses (0% = -100 dBm, 100% = -50 dBm).
func (w WiFiInfo) SignalDBM() int {
	return w.SignalPercent/2 - 100
}

// SameChannelCount returns how many nearby APs share this connection's channel,
// a rough interference indicator.
func (w WiFiInfo) SameChannelCount() int {
	n := 0
	for _, ap := range w.Nearby {
		if ap.Channel == w.Channel {
			n++
		}
	}
	return n
}

// RecommendChannel suggests the least-congested channel for the current band.
//
// ponytail: a count-based heuristic, not an RF planner. On 2.4 GHz it scores the
// non-overlapping channels 1/6/11 (each overlaps its neighbours by ~4). On 5 GHz
// it only considers channels already in use plus the current one, so it never
// suggests a possibly-illegal channel for the region. Returns the current
// channel when it is already the clearest.
func (w WiFiInfo) RecommendChannel() int {
	if w.Channel == 0 {
		return 0
	}
	twoGHz := w.Channel <= 14

	var candidates []int
	if twoGHz {
		candidates = []int{1, 6, 11}
	} else {
		seen := map[int]bool{w.Channel: true}
		candidates = []int{w.Channel}
		for _, ap := range w.Nearby {
			if ap.Channel > 14 && !seen[ap.Channel] {
				seen[ap.Channel] = true
				candidates = append(candidates, ap.Channel)
			}
		}
	}

	span := 0
	if twoGHz {
		span = 4
	}
	best, bestScore := w.Channel, 1<<30
	for _, ch := range candidates {
		score := 0
		for _, ap := range w.Nearby {
			if ap.Channel != ch && abs(ap.Channel-ch) <= span {
				score++
			} else if ap.Channel == ch {
				score++
			}
		}
		// Prefer a strictly better channel; ties keep the current one.
		if score < bestScore || (score == bestScore && ch == w.Channel) {
			best, bestScore = ch, score
		}
	}
	return best
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// AccessPoint is a single visible wireless network.
type AccessPoint struct {
	SSID          string
	Channel       int
	SignalPercent int
}

// WiFiInspector reports the current wireless state, or an error if there is no
// wireless interface.
type WiFiInspector interface {
	WiFi() (WiFiInfo, error)
}
