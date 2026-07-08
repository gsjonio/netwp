// Command netwp is a terminal network manager. This entry point is the
// composition root: it wires concrete adapters into the core use cases.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gsjonio/netwp/internal/adapter/arpscan"
	"github.com/gsjonio/netwp/internal/adapter/httpspeed"
	"github.com/gsjonio/netwp/internal/adapter/netinfo"
	"github.com/gsjonio/netwp/internal/adapter/oui"
	"github.com/gsjonio/netwp/internal/adapter/tcpprobe"
	"github.com/gsjonio/netwp/internal/core"
	"github.com/gsjonio/netwp/internal/tui"
)

const (
	scanTimeout       = 20 * time.Second // one-shot scan budget
	monitorEvery      = 10 * time.Second // interval between monitor scans
	monitorScanBudget = 30 * time.Second // max time a single monitor scan may run
	offlineAfter      = 30 * time.Second // grace before a missing device is offline
	speedtestTimeout  = 30 * time.Second // download + upload budget
)

func main() {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	var err error
	switch command {
	case "", "scan":
		err = runScan()
	case "monitor":
		err = runMonitor()
	case "speedtest":
		err = runSpeedtest()
	default:
		err = fmt.Errorf("unknown command %q (use: scan | monitor | speedtest)", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "netwp:", err)
		os.Exit(1)
	}
}

// buildDiscovery assembles the discovery use case from its platform adapters.
func buildDiscovery() *core.Discovery {
	return core.NewDiscovery(arpscan.New(), netinfo.DNSResolver{}, oui.New(), tcpprobe.New())
}

func runScan() error {
	network, err := netinfo.LocalNetwork()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	var devices []core.Device
	err = withSpinner(fmt.Sprintf("scanning %s", network.CIDR), func() error {
		devices, err = buildDiscovery().Run(ctx, network)
		return err
	})
	if err != nil {
		return err
	}
	tui.RenderDevices(os.Stdout, devices)
	fmt.Printf("\n%d device(s) found.\n", len(devices))
	return nil
}

// withSpinner animates a braille spinner with an elapsed timer on stderr while
// fn runs, so a blocking scan shows progress. stdout stays clean for the table.
func withSpinner(label string, fn func() error) error {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
		start := time.Now()
		for i := 0; ; i++ {
			select {
			case <-done:
				fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", len(label)+16))
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, "\r%c %s… %.1fs", frames[i%len(frames)], label, time.Since(start).Seconds())
			}
		}
	}()
	err := fn()
	close(done)
	return err
}

func runSpeedtest() error {
	ctx, cancel := context.WithTimeout(context.Background(), speedtestTimeout)
	defer cancel()

	var result core.BandwidthResult
	var err error
	err = withSpinner("running speed test", func() error {
		result, err = core.NewSpeedtest(httpspeed.New()).Run(ctx)
		return err
	})
	if err != nil {
		return err
	}
	tui.RenderBandwidth(os.Stdout, result)
	return nil
}

func runMonitor() error {
	network, err := netinfo.LocalNetwork()
	if err != nil {
		return err
	}
	tracker := core.NewTracker(offlineAfter)
	return tui.RunMonitor(buildDiscovery(), tracker, network, monitorEvery, monitorScanBudget)
}
