package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gsjonio/netwp/internal/adapter/eventlog"
	"github.com/gsjonio/netwp/internal/adapter/httpspeed"
	"github.com/gsjonio/netwp/internal/adapter/icmpping"
	"github.com/gsjonio/netwp/internal/adapter/ifacestat"
	"github.com/gsjonio/netwp/internal/adapter/netinfo"
	"github.com/gsjonio/netwp/internal/adapter/wifi"
	"github.com/gsjonio/netwp/internal/core"
	"github.com/gsjonio/netwp/internal/tui"
)

// runMonitor starts the live monitor. quiet runs it headless (no TUI); alertDown
// (bytes/sec, 0 = off) enables the bandwidth-drop alert.
func runMonitor(quiet bool, alertDown float64) error {
	if quiet {
		return runMonitorQuiet()
	}

	discovery, network, err := discoveryContext(nil)
	if err != nil {
		return err
	}
	tracker := core.NewTracker(offlineAfter)

	var reader core.CounterReader
	if alertDown > 0 {
		info, err := netinfo.Interface{}.Inspect()
		if err != nil {
			return err
		}
		reader = ifacestat.New(info.Name)
	}
	return tui.RunMonitor(tui.MonitorConfig{
		Discovery:  discovery,
		Tracker:    tracker,
		Network:    network,
		Interval:   monitorEvery,
		ScanBudget: monitorScanBudget,
		Reader:     reader,
		AlertDown:  alertDown,
		Logger:     defaultEventLogger(),
		Watchlist:  defaultWatchlist(),
	})
}

// runMonitorQuiet runs the same scan/track/log loop as the TUI monitor but with
// no interface: it prints a plain line per join/leave to stdout and persists
// each event, so it can run headless (a systemd/Task Scheduler service, or piped
// to a file). Ctrl-C or SIGTERM stops it cleanly between scans.
func runMonitorQuiet() error {
	discovery, network, err := discoveryContext(nil)
	if err != nil {
		return err
	}
	tracker := core.NewTracker(offlineAfter)
	logger := defaultEventLogger()
	watchlist := defaultWatchlist()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(os.Stderr, "netwp monitor (headless) on %s, scanning every %s — Ctrl-C to stop\n", network.CIDR, monitorEvery)

	ticker := time.NewTicker(monitorEvery)
	defer ticker.Stop()
	for {
		scanCtx, cancel := context.WithTimeout(ctx, monitorScanBudget)
		devices, scanErr := discovery.Run(scanCtx, network)
		cancel()
		if ctx.Err() != nil {
			return nil // stopped mid-scan
		}
		if scanErr != nil {
			fmt.Fprintln(os.Stderr, "scan error:", scanErr)
		} else {
			for _, e := range tracker.Observe(devices, time.Now()) {
				watched := watchlist != nil && watchlist.IsWatched(e.Device.MAC)
				fmt.Println(plainEvent(e, watched))
				if logger != nil {
					_ = logger.Log(e)
				}
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// plainEvent renders an event as an unstyled, timestamped line for headless
// output: no ANSI, since this typically lands in a log file or journal. The
// (unknown)/(watched) tags carry what the TUI shows with color instead.
func plainEvent(e core.Event, watched bool) string {
	name := e.Device.Alias
	if name == "" {
		name = e.Device.Hostname
	}
	if name == "" {
		name = e.Device.IP.String()
	}
	ts := e.At.Format("2006-01-02 15:04:05")
	switch {
	case e.Kind == core.Joined && e.Device.Alias == "":
		return fmt.Sprintf("%s  joined  %s (%s) (unknown)", ts, name, e.Device.IP)
	case e.Kind == core.Joined:
		return fmt.Sprintf("%s  joined  %s (%s)", ts, name, e.Device.IP)
	case e.Kind == core.Left && watched:
		return fmt.Sprintf("%s  left    %s (%s) (watched)", ts, name, e.Device.IP)
	default:
		return fmt.Sprintf("%s  left    %s (%s)", ts, name, e.Device.IP)
	}
}

func runDashboard() error {
	discovery, network, err := discoveryContext(nil)
	if err != nil {
		return err
	}
	info, err := netinfo.Interface{}.Inspect()
	if err != nil {
		return err
	}
	tracker := core.NewTracker(offlineAfter)
	return tui.RunDashboard(tui.DashboardConfig{
		Discovery: discovery,
		Tracker:   tracker,
		Network:   network,
		Info:      info,
		Reader:    ifacestat.New(info.Name),
		WiFi:      wifi.New(),
		Speed:     core.NewSpeedtest(httpspeed.New()),
		Pinger:    icmpping.New(),
		Logger:    defaultEventLogger(),
		Watchlist: defaultWatchlist(),
	})
}

// runDoctor runs a quick connectivity diagnosis (interface, gateway,
// internet, DNS, Wi-Fi) and prints each check with a hint on failure.
func runDoctor(asJSON bool) error {
	info, err := netinfo.Interface{}.Inspect()
	if err != nil {
		return err
	}
	doctor := core.NewDoctor(info, icmpping.New(), netinfo.DNSResolver{}, wifi.New())
	checks := doctor.Run()

	if asJSON {
		return printJSON(doctorJSON(checks))
	}

	failed := 0
	for _, c := range checks {
		mark := "\x1b[32m✓\x1b[0m"
		if !c.OK {
			mark = "\x1b[31m✗\x1b[0m"
			failed++
		}
		fmt.Printf("%s %-10s %s\n", mark, c.Name, c.Detail)
	}
	if failed > 0 {
		fmt.Printf("\n%d check(s) failed. Start with the topmost ✗: a link/gateway problem explains the ones below it.\n", failed)
	} else {
		fmt.Println("\nall good.")
	}
	return nil
}

// runEvents prints the last n recorded presence-change events (newest last).
// device (alias or MAC), when non-empty, restricts the output to one device.
// asJSON emits the entries as a JSON array instead of the human-readable lines.
func runEvents(n int, device string, asJSON bool) error {
	path, err := eventlog.DefaultPath()
	if err != nil {
		return err
	}

	// With a device filter, read the whole history so the match isn't limited
	// to the last n lines; then keep the last n of what matches.
	readN := n
	if device != "" {
		readN = 0
	}
	entries, err := eventlog.Tail(path, readN)
	if err != nil {
		return err
	}
	if device != "" {
		entries = eventlog.FilterByDevice(entries, device, deviceMAC(device))
		if len(entries) > n {
			entries = entries[len(entries)-n:]
		}
	}

	if asJSON {
		return printJSON(eventsForJSON(entries))
	}

	if len(entries) == 0 {
		if device != "" {
			fmt.Printf("no events recorded for %q.\n", device)
		} else {
			fmt.Println("no events recorded yet. Run `netwp monitor` or `netwp dashboard` to start logging.")
		}
		return nil
	}
	for _, e := range entries {
		name := e.Name
		if name == "" {
			name = e.IP
		}
		fmt.Printf("%s  %-6s %s (%s)\n", e.At.Local().Format("2006-01-02 15:04:05"), e.Kind, name, e.IP)
	}
	return nil
}

// eventsForJSON returns entries as a non-nil slice, so `events --json` always
// emits a JSON array (`[]` on an empty history), never `null`. eventlog.Entry
// already carries JSON tags, so no separate DTO is needed.
func eventsForJSON(entries []eventlog.Entry) []eventlog.Entry {
	if entries == nil {
		return []eventlog.Entry{}
	}
	return entries
}

// deviceMAC resolves a device argument to a canonical MAC when possible: a MAC
// literal directly, or an alias name via the alias store. Returns "" if it's
// neither (the filter then matches on the logged Name alone).
func deviceMAC(device string) string {
	if mac, err := net.ParseMAC(device); err == nil {
		return mac.String()
	}
	if store, err := openAliasStore(); err == nil {
		for _, a := range store.List() {
			if strings.EqualFold(a.Name, device) {
				return a.MAC.String()
			}
		}
	}
	return ""
}

// parseRate parses a bits-per-second rate like "50Mbps" or "1.5Gbps" into
// bytes/sec (what core.Rate/RateMeter work in). Longest suffix first, since
// "50Mbps" also ends in "bps".
func parseRate(s string) (float64, error) {
	units := []struct {
		suffix string
		scale  float64
	}{
		{"Gbps", 1e9}, {"Mbps", 1e6}, {"Kbps", 1e3}, {"bps", 1},
	}
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			n, err := strconv.ParseFloat(strings.TrimSuffix(s, u.suffix), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid rate %q: %w", s, err)
			}
			return n * u.scale / 8, nil // bits/sec -> bytes/sec
		}
	}
	return 0, fmt.Errorf("invalid rate %q: expected a suffix like Mbps, Kbps, Gbps, bps", s)
}
