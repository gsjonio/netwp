package core

import (
	"context"
	"net"
	"sort"
	"sync"
	"testing"
)

type fakeScanner struct{ devices []Device }

func (f fakeScanner) Scan(context.Context, Network) ([]Device, error) { return f.devices, nil }

type fakeResolver struct{}

func (fakeResolver) Hostname(ip net.IP) string { return "host-" + ip.String() }

type fakeVendor struct{}

func (fakeVendor) Vendor(net.HardwareAddr) string { return "ACME" }

// fakeAlias returns a nickname only for one specific MAC, to prove enrichment
// binds the alias by MAC.
type fakeAlias struct{ mac string }

func (f fakeAlias) Alias(mac net.HardwareAddr) string {
	if mac.String() == f.mac {
		return "Living Room TV"
	}
	return ""
}

// recordingProber notes which IPs were probed and reports a fixed open-port
// set, so the test can assert self/gateway are skipped.
type recordingProber struct {
	mu     sync.Mutex
	probed []string
	ports  []int
}

func (p *recordingProber) OpenPorts(_ context.Context, ip net.IP) []int {
	p.mu.Lock()
	p.probed = append(p.probed, ip.String())
	p.mu.Unlock()
	return p.ports
}

func TestDiscoverySkipsSelfAndGatewayProbe(t *testing.T) {
	self := net.IPv4(192, 168, 1, 10)
	gateway := net.IPv4(192, 168, 1, 1)
	other := net.IPv4(192, 168, 1, 20)

	otherMAC := net.HardwareAddr{3, 3, 3, 3, 3, 3}
	prober := &recordingProber{ports: []int{portSSH}} // SSH -> ClassComputer
	d := NewDiscovery(
		fakeScanner{devices: []Device{
			{IP: self, MAC: net.HardwareAddr{1, 1, 1, 1, 1, 1}},
			{IP: gateway, MAC: net.HardwareAddr{2, 2, 2, 2, 2, 2}},
			{IP: other, MAC: otherMAC},
		}},
		fakeResolver{}, fakeVendor{}, prober, fakeAlias{mac: otherMAC.String()},
	)

	devices, err := d.Run(context.Background(), Network{Self: self, Gateway: gateway})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sort.Strings(prober.probed)
	if len(prober.probed) != 1 || prober.probed[0] != other.String() {
		t.Errorf("probed = %v, want only %s (self and gateway skipped)", prober.probed, other)
	}

	byIP := map[string]Device{}
	for _, dev := range devices {
		byIP[dev.IP.String()] = dev
	}
	if got := byIP[self.String()].Class; got != ClassThisDevice {
		t.Errorf("self class = %v, want ClassThisDevice", got)
	}
	if got := byIP[gateway.String()].Class; got != ClassRouter {
		t.Errorf("gateway class = %v, want ClassRouter", got)
	}
	if got := byIP[other.String()].Class; got != ClassComputer {
		t.Errorf("other class = %v, want ClassComputer (SSH open)", got)
	}
	if got := byIP[other.String()].Hostname; got != "host-"+other.String() {
		t.Errorf("hostname enrichment missing: %q", got)
	}
	if got := byIP[other.String()].Vendor; got != "ACME" {
		t.Errorf("vendor enrichment missing: %q", got)
	}
	if got := byIP[other.String()].Alias; got != "Living Room TV" {
		t.Errorf("alias enrichment missing: %q", got)
	}
	if got := byIP[self.String()].Alias; got != "" {
		t.Errorf("self should have no alias, got %q", got)
	}
}
