package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gsjonio/netwp/internal/adapter/httpspeed"
	"github.com/gsjonio/netwp/internal/adapter/icmpping"
	"github.com/gsjonio/netwp/internal/adapter/scancache"
	"github.com/gsjonio/netwp/internal/adapter/tcpprobe"
	"github.com/gsjonio/netwp/internal/core"
	"github.com/gsjonio/netwp/internal/tui"
)

func runScan(asJSON, diff bool) error {
	discovery, network, err := discoveryContext()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	var devices []core.Device
	err = withSpinner(fmt.Sprintf("scanning %s", network.CIDR), func() error {
		devices, err = discovery.Run(ctx, network)
		return err
	})
	if err != nil {
		return err
	}

	// Cache path resolved once: --diff reads the previous snapshot from it
	// before Save below overwrites it with this scan's.
	cachePath, cacheErr := scancache.DefaultPath()

	switch {
	case diff:
		var previous []core.Device
		if cacheErr == nil {
			previous, _ = scancache.Load(cachePath)
		}
		printDiff(os.Stdout, core.Diff(previous, devices))
	case asJSON:
		if err := tui.RenderDevicesJSON(os.Stdout, devices); err != nil {
			return err
		}
	default:
		tui.RenderDevices(os.Stdout, devices)
		fmt.Printf("\n%d device(s) found.\n", len(devices))
	}

	// Cache the scan snapshot so `alias set <ip>` and the next `--diff` can
	// skip a fresh scan. Best-effort: a failed write just means the next
	// alias re-scans, and the next --diff has nothing to compare against.
	if cacheErr == nil {
		_ = scancache.Save(cachePath, devices)
	}
	return nil
}

// printDiff writes only what changed since the previous scan snapshot: no
// full device table, since --diff exists precisely to avoid re-reading one.
func printDiff(w io.Writer, d core.DiffResult) {
	if len(d.Joined)+len(d.Left)+len(d.IPChanged)+len(d.MACChanged)+len(d.DupMAC) == 0 {
		fmt.Fprintln(w, "no changes since last scan.")
		return
	}
	for _, dev := range d.Joined {
		fmt.Fprintf(w, "+ %s (%s) joined\n", dev.IP, dev.MAC)
	}
	for _, dev := range d.Left {
		fmt.Fprintf(w, "- %s (%s) left\n", dev.IP, dev.MAC)
	}
	for _, dev := range d.IPChanged {
		fmt.Fprintf(w, "~ %s is now at %s\n", dev.MAC, dev.IP)
	}
	for _, dev := range d.MACChanged {
		fmt.Fprintf(w, "⚠ %s now answers as a different MAC (%s) -- possible address takeover\n", dev.IP, dev.MAC)
	}
	for _, dev := range d.DupMAC {
		fmt.Fprintf(w, "⚠ MAC %s seen at more than one IP this scan (%s)\n", dev.MAC, dev.IP)
	}
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

// runPorts probes a single IP directly: ICMP reachability plus the same
// well-known TCP ports a scan checks for classification, but reported in
// full instead of being folded into a class guess.
func runPorts() error {
	args := os.Args[2:]
	if len(args) < 1 {
		return errors.New("usage: netwp ports <ip>")
	}
	ip := net.ParseIP(args[0])
	if ip == nil {
		return fmt.Errorf("invalid IP %q", args[0])
	}

	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	var open []int
	var rtt time.Duration
	var ttl int
	var reachable bool
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		open = tcpprobe.New().OpenPorts(ctx, ip)
	}()
	go func() {
		defer wg.Done()
		rtt, ttl, reachable = icmpping.New().Ping(ip, 500*time.Millisecond)
	}()
	wg.Wait()

	if reachable {
		fmt.Printf("%s: reachable, RTT %s, TTL %s\n", ip, rtt.Round(time.Millisecond), tui.TTLText(ttl))
	} else {
		fmt.Printf("%s: no ICMP reply\n", ip)
	}
	if len(open) == 0 {
		fmt.Println("no open ports found among the probed set.")
		return nil
	}
	fmt.Println("open ports:")
	for _, p := range open {
		fmt.Printf("  %-6d %s\n", p, portName(p))
	}
	return nil
}

func runSpeedtest() error {
	ctx, cancel := context.WithTimeout(context.Background(), speedtestTimeout)
	defer cancel()

	tester := httpspeed.New()
	var result core.BandwidthResult
	var err error
	err = withSpinner("running speed test", func() error {
		result, err = core.NewSpeedtest(tester).Run(ctx)
		return err
	})
	if err != nil {
		return err
	}
	tui.RenderBandwidth(os.Stdout, result)
	if colo := tester.Colo(ctx); colo != "" {
		fmt.Printf("via Cloudflare edge: %s (nearest of ~300, picked automatically)\n", colo)
	}
	return nil
}
