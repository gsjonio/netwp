// Command netwp is a terminal network manager. This entry point is the
// composition root: it wires concrete adapters into the core use case.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gsjonio/netwp/internal/adapter/arpscan"
	"github.com/gsjonio/netwp/internal/adapter/netinfo"
	"github.com/gsjonio/netwp/internal/adapter/oui"
	"github.com/gsjonio/netwp/internal/core"
	"github.com/gsjonio/netwp/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "netwp:", err)
		os.Exit(1)
	}
}

func run() error {
	network, err := netinfo.LocalNetwork()
	if err != nil {
		return err
	}
	fmt.Printf("Scanning %s from %s ...\n\n", network.CIDR, network.Self)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	discovery := core.NewDiscovery(arpscan.New(), netinfo.DNSResolver{}, oui.New())
	devices, err := discovery.Run(ctx, network)
	if err != nil {
		return err
	}

	tui.RenderDevices(os.Stdout, devices)
	fmt.Printf("\n%d device(s) found.\n", len(devices))
	return nil
}
