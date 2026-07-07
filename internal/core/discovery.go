package core

import (
	"context"
	"sync"
)

// Discovery is the device-discovery use case. It orchestrates a scan and then
// enriches each result with hostname and vendor. It depends only on ports, so
// it is fully testable with fakes.
type Discovery struct {
	scanner Scanner
	names   HostResolver
	vendors VendorLookup
}

func NewDiscovery(scanner Scanner, names HostResolver, vendors VendorLookup) *Discovery {
	return &Discovery{scanner: scanner, names: names, vendors: vendors}
}

// Run scans the target network and returns the enriched devices found.
func (d *Discovery) Run(ctx context.Context, target Network) ([]Device, error) {
	devices, err := d.scanner.Scan(ctx, target)
	if err != nil {
		return nil, err
	}
	// Enrich concurrently: reverse DNS can block up to its timeout per host, so
	// resolving them in parallel keeps a scan bound by the slowest single lookup
	// rather than their sum. Each goroutine writes a distinct slice element.
	var wg sync.WaitGroup
	for i := range devices {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			devices[i].Hostname = d.names.Hostname(devices[i].IP)
			devices[i].Vendor = d.vendors.Vendor(devices[i].MAC)
		}(i)
	}
	wg.Wait()
	return devices, nil
}
