package core

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

type doctorPinger struct{ up map[string]bool }

func (p doctorPinger) Ping(ip net.IP, _ time.Duration) (time.Duration, int, bool) {
	return time.Millisecond, 64, p.up[ip.String()]
}

type doctorNames struct{ ok bool }

func (n doctorNames) Resolve(string) ([]net.IP, error) {
	if !n.ok {
		return nil, errors.New("no such host")
	}
	return []net.IP{net.ParseIP("104.16.0.1")}, nil
}

func check(checks []Check, name string) (Check, bool) {
	for _, c := range checks {
		if c.Name == name {
			return c, true
		}
	}
	return Check{}, false
}

func TestDoctorAllHealthy(t *testing.T) {
	info := InterfaceInfo{Name: "eth0", IP: net.ParseIP("192.168.1.10"), Gateway: net.ParseIP("192.168.1.1"), DHCP: true}
	up := doctorPinger{up: map[string]bool{"192.168.1.1": true, doctorInternetIP: true}}
	d := NewDoctor(info, up, doctorNames{ok: true}, nil)

	for _, c := range d.Run() {
		if !c.OK {
			t.Errorf("check %q failed unexpectedly: %s", c.Name, c.Detail)
		}
	}
}

func TestDoctorNoInternet(t *testing.T) {
	// Gateway up, internet down: the Internet hint should point at the ISP.
	info := InterfaceInfo{Name: "eth0", IP: net.ParseIP("192.168.1.10"), Gateway: net.ParseIP("192.168.1.1")}
	up := doctorPinger{up: map[string]bool{"192.168.1.1": true}} // internet IP absent -> down
	d := NewDoctor(info, up, doctorNames{ok: false}, nil)

	checks := d.Run()
	if gw, _ := check(checks, "Gateway"); !gw.OK {
		t.Error("gateway should be up")
	}
	inet, _ := check(checks, "Internet")
	if inet.OK {
		t.Fatal("internet should be down")
	}
	if !strings.Contains(inet.Detail, "ISP") {
		t.Errorf("internet hint = %q, want it to mention the ISP (gateway was up)", inet.Detail)
	}
}
