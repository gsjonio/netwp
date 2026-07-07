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
		{"windows by port", mk("192.168.0.23", ""), []int{portSMB}, ClassComputer},
		{"iot by vendor", mk("192.168.0.24", "Raspberry Pi Foundation"), nil, ClassIoT},
		{"ambiguous vendor stays unknown", mk("192.168.0.25", "Apple, Inc."), nil, ClassUnknown},
		{"port beats vendor", mk("192.168.0.26", "Raspberry Pi"), []int{portSMB}, ClassComputer},
	}

	for _, c := range cases {
		if got := Classify(c.dev, gw, self, c.ports); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
