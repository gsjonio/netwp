package core

import "context"

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
	for i := range devices {
		devices[i].Hostname = d.names.Hostname(devices[i].IP)
		devices[i].Vendor = d.vendors.Vendor(devices[i].MAC)
	}
	return devices, nil
}
