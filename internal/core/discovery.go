package core

import (
	"context"
	"sync"
	"time"
)

const (
	pingTimeout = 500 * time.Millisecond
	// enrichConcurrency caps how many devices are enriched at once. Each
	// enriched device holds several concurrent sockets (a TCP probe per
	// well-known port, an ICMP ping, and mDNS/NetBIOS lookups), so an
	// unbounded fan-out on a busy LAN could approach the process file-
	// descriptor limit. 32 devices in flight keeps that in the low hundreds.
	enrichConcurrency = 32
)

// Discovery is the device-discovery use case. It orchestrates a scan and then
// enriches each result with hostname, vendor, class and round-trip time. It
// depends only on ports, so it is fully testable with fakes.
type Discovery struct {
	scanner  Scanner
	names    HostResolver
	vendors  VendorLookup
	prober   Prober
	aliases  AliasLookup
	pinger   Pinger
	classes  ClassLookup    // nil disables manual class overrides
	services ServiceScanner // nil disables mDNS service-based classification
}

// DiscoveryDeps are the ports NewDiscovery wires into a Discovery. Named
// fields (over a long positional argument list) keep two same-shaped optional
// interfaces from being transposed silently. Classes and Services are optional:
// a nil field disables that signal.
type DiscoveryDeps struct {
	Scanner  Scanner
	Names    HostResolver
	Vendors  VendorLookup
	Prober   Prober
	Aliases  AliasLookup
	Pinger   Pinger
	Classes  ClassLookup    // nil disables manual class overrides
	Services ServiceScanner // nil disables mDNS service-based classification
}

func NewDiscovery(d DiscoveryDeps) *Discovery {
	return &Discovery{
		scanner:  d.Scanner,
		names:    d.Names,
		vendors:  d.Vendors,
		prober:   d.Prober,
		aliases:  d.Aliases,
		pinger:   d.Pinger,
		classes:  d.Classes,
		services: d.Services,
	}
}

// Run scans the target network and returns the enriched, classified devices.
func (d *Discovery) Run(ctx context.Context, target Network) ([]Device, error) {
	devices, err := d.scanner.Scan(ctx, target)
	if err != nil {
		return nil, err
	}
	// Enrich concurrently across hosts. Within each host, reverse DNS (up to
	// its timeout) and the TCP probe (up to its timeout) are independent, so
	// they run in parallel and the host is bound by max(dns, probe) rather
	// than their sum. Vendor lookup is an in-memory table hit, done inline.
	// Self and the gateway are classified by identity, so their ports are
	// never consulted: skip probing them entirely.
	//
	// ponytail: the whole scan completes before enrichment starts. Streaming
	// scan results into enrichment as they arrive would overlap both phases;
	// worth it only once ranges grow past a /24.
	// Kick off the network-wide mDNS service sweep concurrently with
	// enrichment; each device waits on servicesReady right before Classify, so
	// the sweep's ~1s listen window overlaps the per-host ping/probe work
	// instead of adding to the total. serviceMap is written only before the
	// channel closes and read only after, so the close orders the two safely.
	var serviceMap map[string][]string
	servicesReady := make(chan struct{})
	go func() {
		defer close(servicesReady)
		if d.services != nil {
			serviceMap = d.services.Services(ctx)
		}
	}()

	var wg sync.WaitGroup
	sem := make(chan struct{}, enrichConcurrency)
	for i := range devices {
		wg.Add(1)
		sem <- struct{}{} // block once enrichConcurrency devices are in flight
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			dev := &devices[i]
			skipProbe := dev.IP.Equal(target.Self) ||
				(target.Gateway != nil && dev.IP.Equal(target.Gateway)) ||
				IsLocalMAC(dev.MAC, target.LocalMACs)

			var ports []int
			var inner sync.WaitGroup
			if !skipProbe {
				inner.Add(1)
				go func() {
					defer inner.Done()
					ports = d.prober.OpenPorts(ctx, dev.IP)
				}()
			}
			inner.Add(1)
			go func() {
				defer inner.Done()
				dev.RTT, dev.TTL, dev.Reachable = d.pinger.Ping(dev.IP, pingTimeout)
			}()
			dev.Hostname = d.names.Hostname(dev.IP)
			dev.Vendor = d.vendors.Vendor(dev.MAC)
			dev.Alias = d.aliases.Alias(dev.MAC)
			inner.Wait()
			<-servicesReady // block only if the sweep is still listening
			dev.Ports = ports
			dev.Class = Classify(*dev, target.Gateway, target.Self, ports, target.LocalMACs, serviceMap[dev.IP.String()])
			// A user-pinned class wins over the guess: they told us what this
			// device is, so stop second-guessing it.
			if d.classes != nil {
				if c, ok := d.classes.ClassOverride(dev.MAC); ok {
					dev.Class = c
				}
			}
		}(i)
	}
	wg.Wait()
	return devices, nil
}
