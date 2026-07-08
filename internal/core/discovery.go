package core

import (
	"context"
	"sync"
	"time"
)

const pingTimeout = 500 * time.Millisecond

// Discovery is the device-discovery use case. It orchestrates a scan and then
// enriches each result with hostname, vendor, class and round-trip time. It
// depends only on ports, so it is fully testable with fakes.
type Discovery struct {
	scanner Scanner
	names   HostResolver
	vendors VendorLookup
	prober  Prober
	aliases AliasLookup
	pinger  Pinger
}

func NewDiscovery(scanner Scanner, names HostResolver, vendors VendorLookup, prober Prober, aliases AliasLookup, pinger Pinger) *Discovery {
	return &Discovery{scanner: scanner, names: names, vendors: vendors, prober: prober, aliases: aliases, pinger: pinger}
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
	var wg sync.WaitGroup
	for i := range devices {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			dev := &devices[i]
			skipProbe := dev.IP.Equal(target.Self) ||
				(target.Gateway != nil && dev.IP.Equal(target.Gateway))

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
				dev.RTT, dev.Reachable = d.pinger.Ping(dev.IP, pingTimeout)
			}()
			dev.Hostname = d.names.Hostname(dev.IP)
			dev.Vendor = d.vendors.Vendor(dev.MAC)
			dev.Alias = d.aliases.Alias(dev.MAC)
			inner.Wait()
			dev.Class = Classify(*dev, target.Gateway, target.Self, ports)
		}(i)
	}
	wg.Wait()
	return devices, nil
}
