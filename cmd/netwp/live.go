package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/gsjonio/netwp/internal/adapter/eventlog"
	"github.com/gsjonio/netwp/internal/adapter/httpspeed"
	"github.com/gsjonio/netwp/internal/adapter/icmpping"
	"github.com/gsjonio/netwp/internal/adapter/ifacestat"
	"github.com/gsjonio/netwp/internal/adapter/netinfo"
	"github.com/gsjonio/netwp/internal/adapter/wifi"
	"github.com/gsjonio/netwp/internal/core"
	"github.com/gsjonio/netwp/internal/tui"
)

func runMonitor() error {
	discovery, network, err := discoveryContext(nil)
	if err != nil {
		return err
	}
	tracker := core.NewTracker(offlineAfter)

	alertDown, err := parseAlertFlag(os.Args[2:])
	if err != nil {
		return err
	}
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
func runDoctor() error {
	info, err := netinfo.Interface{}.Inspect()
	if err != nil {
		return err
	}
	doctor := core.NewDoctor(info, icmpping.New(), netinfo.DNSResolver{}, wifi.New())
	checks := doctor.Run()

	if hasArg("--json") {
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

// runEvents prints the last n recorded presence-change events (newest last),
// n defaulting to 20. --device=<alias-or-mac> restricts to one device.
// Usage: netwp events [n] [--device=<alias-or-mac>]
func runEvents() error {
	n := 20
	device := ""
	for _, a := range os.Args[2:] {
		if v, ok := strings.CutPrefix(a, "--device="); ok {
			device = v
			continue
		}
		v, err := strconv.Atoi(a)
		if err != nil || v <= 0 {
			return fmt.Errorf("invalid argument %q: expected a positive count or --device=<alias-or-mac>", a)
		}
		n = v
	}

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

// parseAlertFlag reads "--alert-down=<rate>" out of the monitor subcommand's
// arguments. Returns 0 (alert disabled) when the flag isn't present.
func parseAlertFlag(args []string) (float64, error) {
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, "--alert-down="); ok {
			return parseRate(v)
		}
	}
	return 0, nil
}
