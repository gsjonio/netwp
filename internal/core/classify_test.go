package core

import (
	"net"
	"testing"
)

func TestClassify(t *testing.T) {
	self := net.ParseIP("192.168.0.10").To4()
	gw := net.ParseIP("192.168.0.1").To4()

	mk := func(ip, vendor string) Device {
		return Device{IP: net.ParseIP(ip).To4(), Vendor: vendor}
	}

	cases := []struct {
		name  string
		dev   Device
		ports []int
		want  DeviceClass
	}{
		{"self wins over everything", mk("192.168.0.10", "Netgear"), []int{80}, ClassThisDevice},
		{"gateway is router", mk("192.168.0.1", ""), nil, ClassRouter},
		{"printer by port", mk("192.168.0.20", ""), []int{portPrintRaw}, ClassPrinter},
		{"iphone by port", mk("192.168.0.21", "Apple"), []int{portAppleSync}, ClassMobile},
		{"chromecast by port", mk("192.168.0.22", ""), []int{portChromecast}, ClassMedia},
		{"plex is media", mk("192.168.0.27", ""), []int{portPlex}, ClassMedia},
		{"camera (rtsp) is iot", mk("192.168.0.28", ""), []int{portRTSP}, ClassIoT},
		{"home assistant is iot", mk("192.168.0.29", ""), []int{portHomeAssist}, ClassIoT},
		{"vnc is a computer", mk("192.168.0.30", ""), []int{portVNC}, ClassComputer},
		{"windows by port", mk("192.168.0.23", ""), []int{portSMB}, ClassComputer},
		{"iot by vendor", mk("192.168.0.24", "Raspberry Pi Foundation"), nil, ClassIoT},
		{"ambiguous vendor stays unknown", mk("192.168.0.25", "Apple, Inc."), nil, ClassUnknown},
		{"port beats vendor", mk("192.168.0.26", "Raspberry Pi"), []int{portSMB}, ClassComputer},
	}

	for _, c := range cases {
		if got := Classify(c.dev, gw, self, c.ports, nil); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func TestParseClass(t *testing.T) {
	cases := map[string]DeviceClass{
		"mobile": ClassMobile, "Mobile": ClassMobile, "  IOT ": ClassIoT,
		"computer": ClassComputer, "router": ClassRouter,
		"media": ClassMedia, "printer": ClassPrinter,
	}
	for in, want := range cases {
		if got, ok := ParseClass(in); !ok || got != want {
			t.Errorf("ParseClass(%q) = (%v, %v), want (%v, true)", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "phone", "this device", "unknown"} {
		if _, ok := ParseClass(bad); ok {
			t.Errorf("ParseClass(%q) accepted, want rejected", bad)
		}
	}
}

// TestClassifyLocalMAC covers the multi-homed case: a second interface (e.g.
// Wi-Fi alongside Ethernet) gets a different IP than Self, so it can only be
// recognized as "this device" by MAC.
func TestClassifyLocalMAC(t *testing.T) {
	self := net.ParseIP("192.168.0.10").To4()
	otherNIC := net.HardwareAddr{0x28, 0x0c, 0x50, 0xf4, 0x11, 0x9f}
	dev := Device{IP: net.ParseIP("192.168.0.50").To4(), MAC: otherNIC}

	if got := Classify(dev, nil, self, nil, []net.HardwareAddr{otherNIC}); got != ClassThisDevice {
		t.Errorf("got %v, want ClassThisDevice for a MAC matching one of our own interfaces", got)
	}

	unrelated := net.HardwareAddr{1, 2, 3, 4, 5, 6}
	dev2 := Device{IP: net.ParseIP("192.168.0.51").To4(), MAC: unrelated}
	if got := Classify(dev2, nil, self, nil, []net.HardwareAddr{otherNIC}); got == ClassThisDevice {
		t.Errorf("got ClassThisDevice for an unrelated MAC, want no false positive")
	}
}
