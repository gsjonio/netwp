package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/gsjonio/netwp/internal/adapter/netinfo"
	"github.com/gsjonio/netwp/internal/core"
	"github.com/gsjonio/netwp/internal/tui"
)

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
	return promptYes()
}

// promptYes reads one line from stdin and reports whether it is exactly "yes".
func promptYes() bool {
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line) == "yes"
}
