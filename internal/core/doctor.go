package core

import (
	"fmt"
	"net"
	"time"
)

// NameChecker resolves a hostname to IPs, to confirm forward DNS works.
type NameChecker interface {
	Resolve(host string) ([]net.IP, error)
}

// Check is one diagnostic result from Doctor.
type Check struct {
	Name   string
	OK     bool
	Detail string // status detail, or a hint on failure
}

const (
	doctorPingTimeout = 1 * time.Second
	doctorDNSProbe    = "cloudflare.com" // a name that should always resolve
	doctorInternetIP  = "1.1.1.1"        // an always-on anycast host to ping
)

// Doctor runs a short sequence of connectivity checks: interface, gateway,
// internet, DNS, and Wi-Fi. Pure orchestration over ports, so it is testable
// with fakes. Each check's hint is tailored using earlier results (e.g. "DNS
// fails but the internet is up" points at the DNS server, not the link).
type Doctor struct {
	info   InterfaceInfo
	pinger Pinger
	names  NameChecker
	wifi   WiFiInspector // may be nil (e.g. a wired-only build/host)
}

func NewDoctor(info InterfaceInfo, pinger Pinger, names NameChecker, wifi WiFiInspector) *Doctor {
	return &Doctor{info: info, pinger: pinger, names: names, wifi: wifi}
}

// Run executes the checks in order and returns their results.
func (d *Doctor) Run() []Check {
	var checks []Check

	// Interface has an IP.
	if d.info.IP != nil {
		how := "static"
		if d.info.DHCP {
			how = "DHCP"
		}
		checks = append(checks, Check{"Interface", true, fmt.Sprintf("%s has %s (%s)", d.info.Name, d.info.IP, how)})
	} else {
		checks = append(checks, Check{"Interface", false, "no IP address -- is the interface up and connected?"})
	}

	// Gateway reachable.
	gatewayUp := false
	if d.info.Gateway == nil {
		checks = append(checks, Check{"Gateway", false, "no gateway configured"})
	} else if rtt, _, ok := d.pinger.Ping(d.info.Gateway, doctorPingTimeout); ok {
		gatewayUp = true
		checks = append(checks, Check{"Gateway", true, fmt.Sprintf("%s responds in %s", d.info.Gateway, rtt.Round(time.Millisecond))})
	} else {
		checks = append(checks, Check{"Gateway", false, fmt.Sprintf("%s not responding -- check the router or the cable", d.info.Gateway)})
	}

	// Internet reachable.
	internetUp := false
	if rtt, _, ok := d.pinger.Ping(net.ParseIP(doctorInternetIP), doctorPingTimeout); ok {
		internetUp = true
		checks = append(checks, Check{"Internet", true, fmt.Sprintf("%s reachable in %s", doctorInternetIP, rtt.Round(time.Millisecond))})
	} else {
		hint := "no route to the internet"
		if gatewayUp {
			hint = "gateway is up but the internet isn't reachable -- likely an ISP or modem issue"
		}
		checks = append(checks, Check{"Internet", false, hint})
	}

	// DNS resolves.
	if ips, err := d.names.Resolve(doctorDNSProbe); err == nil && len(ips) > 0 {
		checks = append(checks, Check{"DNS", true, fmt.Sprintf("%s resolves (%s)", doctorDNSProbe, ips[0])})
	} else {
		hint := fmt.Sprintf("can't resolve %s -- check your DNS servers", doctorDNSProbe)
		if internetUp {
			hint = fmt.Sprintf("internet is up but DNS can't resolve %s -- a DNS server problem", doctorDNSProbe)
		}
		checks = append(checks, Check{"DNS", false, hint})
	}

	// Wi-Fi is informational: a wired host is fine, so this never fails.
	if d.wifi != nil {
		if info, err := d.wifi.WiFi(); err == nil && info.Connected {
			detail := fmt.Sprintf("%s, signal %d%%", info.SSID, info.SignalPercent)
			if info.SignalPercent < 40 {
				detail += " (weak -- try moving closer to the router)"
			}
			checks = append(checks, Check{"Wi-Fi", true, detail})
		} else {
			checks = append(checks, Check{"Wi-Fi", true, "not on Wi-Fi (wired connection)"})
		}
	}

	return checks
}
