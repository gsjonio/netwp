package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
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
	ports, err := portsFlag(os.Args[2:])
	if err != nil {
		return err
	}
	// Validate --class before scanning, so a typo fails fast instead of after a
	// full scan. Applied to the displayed set further down.
	class, filtered, err := classFilter(os.Args[2:])
	if err != nil {
		return err
	}
	discovery, network, err := discoveryContext(ports)
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

	// --class filters the displayed set only; the cache below still stores the
	// full snapshot so alias/--diff keep working against every device.
	shown := devices
	if filtered {
		shown = filterByClass(devices, class)
	}

	switch {
	case diff:
		var previous []core.Device
		if cacheErr == nil {
			previous, _ = scancache.Load(cachePath)
		}
		printDiff(os.Stdout, core.Diff(previous, devices))
	case asJSON:
		if err := tui.RenderDevicesJSON(os.Stdout, shown); err != nil {
			return err
		}
	default:
		tui.RenderDevices(os.Stdout, shown)
		if filtered {
			fmt.Printf("\n%d of %d device(s) match class %q.\n", len(shown), len(devices), class)
		} else {
			fmt.Printf("\n%d device(s) found.\n", len(devices))
		}
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

// portsFlag reads "--ports=<list>" from the scan arguments, returning the
// custom port set or nil (default set) when the flag is absent.
func portsFlag(args []string) ([]int, error) {
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, "--ports="); ok {
			return parsePorts(v)
		}
	}
	return nil, nil
}

// classFilter reads "--class=<name>" from the scan arguments. Returns
// (class, false, nil) when the flag is absent, and an error when the name isn't
// one of the known classes, so a typo fails loudly instead of silently showing
// nothing.
func classFilter(args []string) (core.DeviceClass, bool, error) {
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, "--class="); ok {
			class, ok := core.ParseClass(v)
			if !ok {
				return 0, false, fmt.Errorf("unknown class %q: expected one of router, computer, mobile, media, printer, iot", v)
			}
			return class, true, nil
		}
	}
	return 0, false, nil
}

// filterByClass keeps only devices of the given class.
func filterByClass(devices []core.Device, class core.DeviceClass) []core.Device {
	out := make([]core.Device, 0, len(devices))
	for _, d := range devices {
		if d.Class == class {
			out = append(out, d)
		}
	}
	return out
}

// parsePorts turns "22,80,443" into a port slice. Comma-separated individual
// ports only (no ranges): the whole point of the curated default is to avoid a
// full sweep, and listing ports keeps that friction while allowing an extra one
// or two (a dev server on 3000, say).
func parsePorts(s string) ([]int, error) {
	var ports []int
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		p, err := strconv.Atoi(tok)
		if err != nil || p < 1 || p > 65535 {
			return nil, fmt.Errorf("invalid port %q: expected 1-65535", tok)
		}
		ports = append(ports, p)
	}
	if len(ports) == 0 {
		return nil, errors.New("--ports needs at least one port, e.g. --ports=22,80,443")
	}
	return ports, nil
}

// portsTargetIP returns the first non-flag argument parsed as an IP, so `netwp
// ports <ip> --json` and `netwp ports --json <ip>` both work.
func portsTargetIP(args []string) (net.IP, error) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		ip := net.ParseIP(a)
		if ip == nil {
			return nil, fmt.Errorf("invalid IP %q", a)
		}
		return ip, nil
	}
	return nil, errors.New("usage: netwp ports <ip> [--json]")
}

// runPorts probes a single IP directly: ICMP reachability plus the same
// well-known TCP ports a scan checks for classification, but reported in
// full instead of being folded into a class guess.
func runPorts() error {
	ip, err := portsTargetIP(os.Args[2:])
	if err != nil {
		return err
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

	if hasArg("--json") {
		result := portsResultJSON{IP: ip.String(), Reachable: reachable, Ports: make([]portJSON, 0, len(open))}
		if reachable {
			ms := float64(rtt.Microseconds()) / 1000
			result.RTTMillis = &ms
			result.TTL = ttl
		}
		for _, p := range open {
			result.Ports = append(result.Ports, portJSON{Port: p, Name: portName(p)})
		}
		return printJSON(result)
	}

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
	edge := tester.Colo(ctx)
	if hasArg("--json") {
		return printJSON(speedtestResultJSON{
			DownloadMbps: result.DownloadMbps,
			UploadMbps:   result.UploadMbps,
			Edge:         edge,
		})
	}
	tui.RenderBandwidth(os.Stdout, result)
	if edge != "" {
		fmt.Printf("via Cloudflare edge: %s (nearest of ~300, picked automatically)\n", edge)
	}
	return nil
}
