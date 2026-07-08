// Command netwp is a terminal network manager. This entry point is the
// composition root: it wires concrete adapters into the core use cases.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gsjonio/netwp/internal/adapter/aliasstore"
	"github.com/gsjonio/netwp/internal/adapter/arpscan"
	"github.com/gsjonio/netwp/internal/adapter/httpspeed"
	"github.com/gsjonio/netwp/internal/adapter/netinfo"
	"github.com/gsjonio/netwp/internal/adapter/oui"
	"github.com/gsjonio/netwp/internal/adapter/scancache"
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
	case "iface":
		err = runIface()
	case "alias":
		err = runAlias()
	default:
		err = fmt.Errorf("unknown command %q (use: scan | monitor | speedtest | iface | alias)", command)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "netwp:", err)
		os.Exit(1)
	}
}

// buildDiscovery assembles the discovery use case from its platform adapters.
func buildDiscovery(aliases core.AliasLookup) *core.Discovery {
	return core.NewDiscovery(arpscan.New(), netinfo.DNSResolver{}, oui.New(), tcpprobe.New(), aliases)
}

// openAliasStore opens the persistent nickname store at its default path.
func openAliasStore() (*aliasstore.Store, error) {
	path, err := aliasstore.DefaultPath()
	if err != nil {
		return nil, err
	}
	return aliasstore.Open(path)
}

func runScan() error {
	network, err := netinfo.LocalNetwork()
	if err != nil {
		return err
	}
	store, err := openAliasStore()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	var devices []core.Device
	err = withSpinner(fmt.Sprintf("scanning %s", network.CIDR), func() error {
		devices, err = buildDiscovery(store).Run(ctx, network)
		return err
	})
	if err != nil {
		return err
	}
	tui.RenderDevices(os.Stdout, devices)
	fmt.Printf("\n%d device(s) found.\n", len(devices))

	// Cache the IP-to-MAC map so `alias set <ip>` can skip a fresh scan.
	// Best-effort: a failed write just means the next alias re-scans.
	if path, err := scancache.DefaultPath(); err == nil {
		_ = scancache.Save(path, devices)
	}
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

// runIface dispatches "iface" (inspect) and its "static"/"dhcp" subcommands.
func runIface() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return runIfaceInspect()
	}
	switch args[0] {
	case "static":
		return runIfaceSetStatic(args[1:])
	case "dhcp":
		return runIfaceSetDHCP()
	default:
		return fmt.Errorf("unknown iface subcommand %q (use: iface | iface static <ip>/<bits> <gateway> [dns...] | iface dhcp)", args[0])
	}
}

func runIfaceInspect() error {
	info, err := netinfo.Interface{}.Inspect()
	if err != nil {
		return err
	}
	tui.RenderInterface(os.Stdout, info)
	return nil
}

// parseStaticArgs parses "iface static" arguments into a StaticConfig. Pure
// and side-effect-free so it can be tested without touching the network.
func parseStaticArgs(args []string) (core.StaticConfig, error) {
	if len(args) < 2 {
		return core.StaticConfig{}, fmt.Errorf("usage: netwp iface static <ip>/<bits> <gateway> [dns...]")
	}
	ip, ipnet, err := net.ParseCIDR(args[0])
	if err != nil {
		return core.StaticConfig{}, fmt.Errorf("invalid address %q: %w", args[0], err)
	}
	gateway := net.ParseIP(args[1])
	if gateway == nil {
		return core.StaticConfig{}, fmt.Errorf("invalid gateway %q", args[1])
	}
	var dns []net.IP
	for _, s := range args[2:] {
		d := net.ParseIP(s)
		if d == nil {
			return core.StaticConfig{}, fmt.Errorf("invalid DNS server %q", s)
		}
		dns = append(dns, d)
	}
	return core.StaticConfig{IP: ip, Mask: net.IP(ipnet.Mask), Gateway: gateway, DNS: dns}, nil
}

func runIfaceSetStatic(args []string) error {
	cfg, err := parseStaticArgs(args)
	if err != nil {
		return err
	}
	if !confirm(fmt.Sprintf("set a static address: %s / %s, gateway %s", cfg.IP, cfg.Mask, cfg.Gateway)) {
		fmt.Println("aborted.")
		return nil
	}
	return netinfo.Configurator{}.SetStatic(cfg)
}

func runIfaceSetDHCP() error {
	if !confirm("switch the active interface back to DHCP") {
		fmt.Println("aborted.")
		return nil
	}
	return netinfo.Configurator{}.SetDHCP()
}

// confirm asks the user to type "yes" before a real, live network change.
// Interface configuration is destructive-ish (can cut the machine off the
// network), so this always asks, with no --yes flag to skip it.
func confirm(action string) bool {
	fmt.Printf("about to %s. This changes your machine's real network config.\nType \"yes\" to continue: ", action)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line) == "yes"
}

func runMonitor() error {
	network, err := netinfo.LocalNetwork()
	if err != nil {
		return err
	}
	store, err := openAliasStore()
	if err != nil {
		return err
	}
	tracker := core.NewTracker(offlineAfter)
	return tui.RunMonitor(buildDiscovery(store), tracker, network, monitorEvery, monitorScanBudget)
}

// runAlias dispatches the alias subcommands: set, ls, rm.
func runAlias() error {
	args := os.Args[2:]
	if len(args) == 0 {
		return errors.New("usage: netwp alias set <ip-or-mac> <name> | alias ls | alias rm <ip-or-mac>")
	}
	store, err := openAliasStore()
	if err != nil {
		return err
	}
	switch args[0] {
	case "ls", "list":
		return runAliasList(store)
	case "set":
		return runAliasSet(store, args[1:])
	case "rm", "remove", "del":
		return runAliasRemove(store, args[1:])
	default:
		return fmt.Errorf("unknown alias subcommand %q (use: set | ls | rm)", args[0])
	}
}

func runAliasList(store *aliasstore.Store) error {
	list := store.List()
	if len(list) == 0 {
		fmt.Println("no aliases set.")
		return nil
	}
	for _, a := range list {
		fmt.Printf("%-17s  %s\n", a.MAC, a.Name)
	}
	return nil
}

func runAliasSet(store *aliasstore.Store, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: netwp alias set <ip-or-mac> <name>")
	}
	mac, err := resolveMAC(args[0])
	if err != nil {
		return err
	}
	name := strings.Join(args[1:], " ")
	if err := store.Set(mac, name); err != nil {
		return err
	}
	fmt.Printf("aliased %s → %q\n", mac, name)
	return nil
}

func runAliasRemove(store *aliasstore.Store, args []string) error {
	if len(args) < 1 {
		return errors.New("usage: netwp alias rm <ip-or-mac>")
	}
	mac, err := resolveMAC(args[0])
	if err != nil {
		return err
	}
	if err := store.Delete(mac); err != nil {
		return err
	}
	fmt.Printf("removed alias for %s\n", mac)
	return nil
}

// resolveMAC turns a CLI argument into a MAC. A MAC literal is used directly.
// An IP is looked up in the last scan's cache first, and only if that misses is
// a fresh ARP sweep run (whose result then refreshes the cache). Keying aliases
// by MAC keeps them stable when DHCP hands the device a new IP.
func resolveMAC(arg string) (net.HardwareAddr, error) {
	if mac, err := net.ParseMAC(arg); err == nil {
		return mac, nil
	}
	ip := net.ParseIP(arg)
	if ip == nil {
		return nil, fmt.Errorf("%q is neither a MAC nor an IP address", arg)
	}

	cachePath, _ := scancache.DefaultPath()
	if cachePath != "" {
		if mac, ok := scancache.Lookup(cachePath, ip); ok {
			return mac, nil
		}
	}

	network, err := netinfo.LocalNetwork()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), scanTimeout)
	defer cancel()

	var devices []core.Device
	err = withSpinner("resolving "+arg, func() error {
		devices, err = arpscan.New().Scan(ctx, network)
		return err
	})
	if err != nil {
		return nil, err
	}
	if cachePath != "" {
		_ = scancache.Save(cachePath, devices)
	}
	for _, d := range devices {
		if d.IP.Equal(ip) {
			return d.MAC, nil
		}
	}
	return nil, fmt.Errorf("no device with IP %s found on the network (is it online?)", arg)
}
