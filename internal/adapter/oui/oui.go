// Package oui resolves a MAC address to its manufacturer.
package oui

import (
	"fmt"
	"net"
	"strings"
)

// Lookup implements core.VendorLookup using an in-memory OUI table.
type Lookup struct{}

func New() Lookup { return Lookup{} }

// ponytail: starter table of common vendors only. For real coverage load the
// full IEEE OUI registry (~35k entries) — download it and embed with go:embed,
// then parse into this map at init. Kept tiny here to stay dependency-free.
var vendors = map[string]string{
	"FCFBFB": "Apple",
	"F0189E": "Apple",
	"3C0754": "Apple",
	"001A11": "Google",
	"D8EB46": "Google",
	"F4F5D8": "Google",
	"DCA632": "Raspberry Pi",
	"B827EB": "Raspberry Pi",
	"E45F01": "Raspberry Pi",
	"001788": "Philips Hue",
	"ECFABC": "Amazon (Echo/Fire)",
	"FCA667": "Amazon (Echo/Fire)",
	"D0038C": "Xiaomi",
	"64CC2E": "Xiaomi",
	"AC2DA9": "TP-Link",
	"50C7BF": "TP-Link",
	"001560": "Samsung",
	"F008D1": "Samsung",
}

// Vendor returns the manufacturer for mac, or "Unknown" if the OUI is unlisted.
func (Lookup) Vendor(mac net.HardwareAddr) string {
	if len(mac) < 3 {
		return ""
	}
	prefix := strings.ToUpper(fmt.Sprintf("%02X%02X%02X", mac[0], mac[1], mac[2]))
	if v, ok := vendors[prefix]; ok {
		return v
	}
	return "Unknown"
}
