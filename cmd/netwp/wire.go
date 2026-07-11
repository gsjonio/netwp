package main

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/gsjonio/netwp/internal/adapter/aliasstore"
	"github.com/gsjonio/netwp/internal/adapter/arpscan"
	"github.com/gsjonio/netwp/internal/adapter/classstore"
	"github.com/gsjonio/netwp/internal/adapter/eventlog"
	"github.com/gsjonio/netwp/internal/adapter/icmpping"
	"github.com/gsjonio/netwp/internal/adapter/namelookup"
	"github.com/gsjonio/netwp/internal/adapter/netinfo"
	"github.com/gsjonio/netwp/internal/adapter/oui"
	"github.com/gsjonio/netwp/internal/adapter/scancache"
	"github.com/gsjonio/netwp/internal/adapter/tcpprobe"
	"github.com/gsjonio/netwp/internal/adapter/watchstore"
	"github.com/gsjonio/netwp/internal/core"
)

// buildDiscovery assembles the discovery use case from its platform adapters.
func buildDiscovery(aliases core.AliasLookup, classes core.ClassLookup) *core.Discovery {
	return core.NewDiscovery(core.DiscoveryDeps{
		Scanner:  arpscan.New(),
		Names:    namelookup.New(netinfo.DNSResolver{}),
		Vendors:  oui.New(),
		Prober:   tcpprobe.New(),
		Aliases:  aliases,
		Pinger:   icmpping.New(),
		Classes:  classes,
		Services: namelookup.NewServiceScanner(),
	})
}

// openAliasStore opens the persistent nickname store at its default path.
func openAliasStore() (*aliasstore.Store, error) {
	path, err := aliasstore.DefaultPath()
	if err != nil {
		return nil, err
	}
	return aliasstore.Open(path)
}

// openClassStore opens the persistent class-override store at its default path.
func openClassStore() (*classstore.Store, error) {
	path, err := classstore.DefaultPath()
	if err != nil {
		return nil, err
	}
	return classstore.Open(path)
}

// discoveryContext resolves the two things every scanning command needs: the
// local network to sweep, and a Discovery wired to the user's alias store.
func discoveryContext() (*core.Discovery, core.Network, error) {
	network, err := netinfo.LocalNetwork()
	if err != nil {
		return nil, core.Network{}, err
	}
	store, err := openAliasStore()
	if err != nil {
		return nil, core.Network{}, err
	}
	classes, err := openClassStore()
	if err != nil {
		return nil, core.Network{}, err
	}
	return buildDiscovery(store, classes), network, nil
}

// defaultEventLogger builds the events.jsonl logger for monitor/dashboard.
// Returns nil (persistence disabled) if the config directory can't be
// resolved -- the same best-effort posture as scancache's writes.
func defaultEventLogger() core.EventLogger {
	path, err := eventlog.DefaultPath()
	if err != nil {
		return nil
	}
	return eventlog.New(path)
}

// defaultWatchlist opens the persistent watch list for monitor/dashboard.
// Returns a nil interface (alerts disabled) if it can't be loaded, so the
// caller's nil check stays correct.
func defaultWatchlist() core.Watchlist {
	path, err := watchstore.DefaultPath()
	if err != nil {
		return nil
	}
	store, err := watchstore.Open(path)
	if err != nil {
		return nil
	}
	return store
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

// resolveWakeTarget resolves a MAC, IP, or alias name to a MAC. Unlike
// resolveMAC it also accepts an alias name (reverse alias lookup), since waking
// a device by its nickname is the common case and the device is likely offline.
func resolveWakeTarget(arg string) (net.HardwareAddr, error) {
	if mac, err := net.ParseMAC(arg); err == nil {
		return mac, nil
	}
	if net.ParseIP(arg) != nil {
		return resolveMAC(arg) // IP: cache first, then ARP (see resolveMAC)
	}
	store, err := openAliasStore()
	if err != nil {
		return nil, err
	}
	for _, a := range store.List() {
		if strings.EqualFold(a.Name, arg) {
			return a.MAC, nil
		}
	}
	return nil, fmt.Errorf("%q is not a MAC, IP, or known alias name", arg)
}
