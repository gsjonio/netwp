package core

import (
	"bytes"
	"net"
	"strings"
)

// DeviceClass is a coarse, best-effort guess at what a host is.
type DeviceClass int

const (
	ClassUnknown    DeviceClass = iota
	ClassThisDevice             // the machine running netwp
	ClassRouter                 // the default gateway
	ClassComputer               // PC / server / SBC
	ClassMobile                 // phone / tablet
	ClassMedia                  // TV / streaming stick / speaker
	ClassPrinter
	ClassIoT // smart home / embedded
)

// ParseClass maps a user-typed class name (as shown by String, case-
// insensitive) to a DeviceClass, for `netwp class set`. Only the classes a
// user would meaningfully pin are accepted: not ThisDevice/Unknown.
func ParseClass(s string) (DeviceClass, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "router":
		return ClassRouter, true
	case "computer":
		return ClassComputer, true
	case "mobile":
		return ClassMobile, true
	case "media":
		return ClassMedia, true
	case "printer":
		return ClassPrinter, true
	case "iot":
		return ClassIoT, true
	default:
		return ClassUnknown, false
	}
}

func (c DeviceClass) String() string {
	switch c {
	case ClassThisDevice:
		return "This device"
	case ClassRouter:
		return "Router"
	case ClassComputer:
		return "Computer"
	case ClassMobile:
		return "Mobile"
	case ClassMedia:
		return "Media"
	case ClassPrinter:
		return "Printer"
	case ClassIoT:
		return "IoT"
	default:
		return "Unknown"
	}
}

// Well-known TCP ports used as classification hints.
const (
	portSSH        = 22
	portAFP        = 548 // Apple file sharing (a Mac)
	portSMB        = 445
	portRDP        = 3389
	portVNC        = 5900
	portMySQL      = 3306
	portPostgres   = 5432
	portPrintRaw   = 9100 // JetDirect
	portPrintLPD   = 515
	portIPP        = 631
	portChromecast = 8009
	portPlex       = 32400
	portJellyfin   = 8096
	portRTSP       = 554  // IP camera
	portMQTT       = 1883 // smart-home broker
	portHomeAssist = 8123
	portAppleSync  = 62078 // iPhone/iPad "lockdownd"
)

// Classify guesses a device's class from the signals a scan can gather: whether
// it is us, whether it is the gateway, which TCP ports answered, and its vendor.
//
// ponytail: pure heuristic, deliberately conservative — identity signals (self,
// gateway) win, then port fingerprints, then a vendor-keyword fallback. Wrong
// guesses fall back to Unknown rather than asserting nonsense.
func Classify(d Device, gateway, self net.IP, openPorts []int, localMACs []net.HardwareAddr) DeviceClass {
	if self != nil && d.IP.Equal(self) {
		return ClassThisDevice
	}
	if IsLocalMAC(d.MAC, localMACs) {
		return ClassThisDevice
	}
	if gateway != nil && d.IP.Equal(gateway) {
		return ClassRouter
	}

	has := func(port int) bool {
		for _, p := range openPorts {
			if p == port {
				return true
			}
		}
		return false
	}
	switch {
	case has(portPrintRaw) || has(portPrintLPD) || has(portIPP):
		return ClassPrinter
	case has(portAppleSync):
		return ClassMobile
	case has(portChromecast) || has(portPlex) || has(portJellyfin):
		return ClassMedia
	case has(portRTSP) || has(portMQTT) || has(portHomeAssist):
		return ClassIoT
	case has(portSMB) || has(portRDP) || has(portVNC) || has(portAFP) || has(portMySQL) || has(portPostgres):
		return ClassComputer
	case has(portSSH):
		return ClassComputer
	}
	return classFromVendor(d.Vendor)
}

// classFromVendor maps distinctive OUI vendors to a likely class. Ambiguous
// vendors (e.g. Apple, Samsung — could be phone, PC or TV) stay Unknown so a
// port fingerprint or nothing decides instead of a coin flip.
// IsLocalMAC reports whether mac belongs to one of this machine's own
// interfaces (e.g. Ethernet and Wi-Fi both connected at once show up as two
// separate devices in a scan; this recognizes the second one as "us" too,
// since its IP won't match Network.Self).
func IsLocalMAC(mac net.HardwareAddr, localMACs []net.HardwareAddr) bool {
	if len(mac) == 0 {
		return false
	}
	for _, local := range localMACs {
		if bytes.Equal(mac, local) {
			return true
		}
	}
	return false
}

func classFromVendor(vendor string) DeviceClass {
	v := strings.ToLower(vendor)
	switch {
	case containsAny(v, "raspberry", "espressif", "tuya", "sonoff", "shelly", "tp-link", "sonos", "ring", "nest"):
		return ClassIoT
	case containsAny(v, "amazon", "roku", "vizio", "chromecast"):
		return ClassMedia
	case containsAny(v, "mikrotik", "netgear", "ubiquiti", "d-link", "mitrastar", "zte", "arris"):
		return ClassRouter
	case containsAny(v, "hewlett", "canon", "epson", "brother", "lexmark"):
		return ClassPrinter
	default:
		return ClassUnknown
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
