package core

import (
	"context"
	"net"
	"sort"
	"sync"
	"testing"
	"time"
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

type fakePinger struct{}

func (fakePinger) Ping(net.IP, time.Duration) (time.Duration, int, bool) {
	return 3 * time.Millisecond, 64, true
}

// fakeClassLookup pins one specific MAC to a class, to prove a manual override
// wins over the automatic guess.
type fakeClassLookup struct {
	mac   string
	class DeviceClass
}

func (f fakeClassLookup) ClassOverride(mac net.HardwareAddr) (DeviceClass, bool) {
	if mac.String() == f.mac {
		return f.class, true
	}
	return ClassUnknown, false
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

// concurrencyResolver tracks the peak number of overlapping Hostname calls.
// Hostname runs on the per-device enrichment goroutine for the whole time
// that device is being enriched, so its peak concurrency is the enrichment
// fan-out width.
type concurrencyResolver struct {
	mu       sync.Mutex
	cur, max int
}

func (c *concurrencyResolver) Hostname(net.IP) string {
	c.mu.Lock()
	c.cur++
	if c.cur > c.max {
		c.max = c.cur
	}
	c.mu.Unlock()
	time.Sleep(2 * time.Millisecond)
	c.mu.Lock()
	c.cur--
	c.mu.Unlock()
	return ""
}

func TestDiscoveryBoundsEnrichmentConcurrency(t *testing.T) {
	var devs []Device
	for i := 0; i < 100; i++ {
		devs = append(devs, Device{IP: net.IPv4(10, 0, 0, byte(i)), MAC: net.HardwareAddr{1, 2, 3, 4, 5, byte(i)}})
	}
	res := &concurrencyResolver{}
	d := NewDiscovery(fakeScanner{devices: devs}, res, fakeVendor{}, &recordingProber{}, fakeAlias{}, fakePinger{}, nil)

	if _, err := d.Run(context.Background(), Network{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.max > enrichConcurrency {
		t.Errorf("peak enrichment concurrency = %d, want <= %d (fan-out must be bounded)", res.max, enrichConcurrency)
	}
	if res.max < 2 {
		t.Errorf("peak concurrency = %d, expected the work to actually overlap", res.max)
	}
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
		fakeResolver{}, fakeVendor{}, prober, fakeAlias{mac: otherMAC.String()}, fakePinger{}, nil,
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
	if d := byIP[other.String()]; !d.Reachable || d.RTT != 3*time.Millisecond || d.TTL != 64 {
		t.Errorf("ping enrichment missing: reachable=%v rtt=%v ttl=%v", d.Reachable, d.RTT, d.TTL)
	}
}

// TestDiscoveryClassOverride proves a user-pinned class replaces the automatic
// guess: SSH open would classify as Computer, but the override says Mobile.
func TestDiscoveryClassOverride(t *testing.T) {
	mac := net.HardwareAddr{9, 9, 9, 9, 9, 9}
	ip := net.IPv4(192, 168, 1, 55)
	d := NewDiscovery(
		fakeScanner{devices: []Device{{IP: ip, MAC: mac}}},
		fakeResolver{}, fakeVendor{}, &recordingProber{ports: []int{portSSH}}, fakeAlias{}, fakePinger{},
		fakeClassLookup{mac: mac.String(), class: ClassMobile},
	)
	devices, err := d.Run(context.Background(), Network{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := devices[0].Class; got != ClassMobile {
		t.Errorf("class = %v, want ClassMobile (override beats the SSH->Computer guess)", got)
	}
}
