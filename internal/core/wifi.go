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
