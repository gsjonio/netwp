package core

import (
	"context"
	"sync"
)

// Discovery is the device-discovery use case. It orchestrates a scan and then
// enriches each result with hostname, vendor and a class guess. It depends only
// on ports, so it is fully testable with fakes.
type Discovery struct {
	scanner Scanner
	names   HostResolver
	vendors VendorLookup
	prober  Prober
}

func NewDiscovery(scanner Scanner, names HostResolver, vendors VendorLookup, prober Prober) *Discovery {
	return &Discovery{scanner: scanner, names: names, vendors: vendors, prober: prober}
}

// Run scans the target network and returns the enriched, classified devices.
func (d *Discovery) Run(ctx context.Context, target Network) ([]Device, error) {
	devices, err := d.scanner.Scan(ctx, target)
	if err != nil {
		return nil, err
	}
	// Enrich concurrently: reverse DNS and port probing each block up to their
	// timeout per host, so doing them in parallel keeps a scan bound by the
	// slowest single host rather than their sum. Each goroutine writes a
	// distinct slice element.
	var wg sync.WaitGroup
	for i := range devices {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			devices[i].Hostname = d.names.Hostname(devices[i].IP)
			devices[i].Vendor = d.vendors.Vendor(devices[i].MAC)
			ports := d.prober.OpenPorts(ctx, devices[i].IP)
			devices[i].Class = Classify(devices[i], target.Gateway, target.Self, ports)
		}(i)
	}
	wg.Wait()
	return devices, nil
}
