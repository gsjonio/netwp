//go:build windows

package netinfo

import (
	"fmt"
	"os/exec"

	"github.com/gsjonio/netwp/internal/core"
)

// Configurator applies IP configuration to the active interface via netsh.
//
// ponytail: shells out to netsh instead of the native SetIPInterfaceEntry /
// CreateUnicastIpAddressEntry APIs. This runs once per deliberate user
// action, not in a hot path, so the extra process spawn doesn't matter and
// it keeps this file short. Arguments are passed as separate exec.Command
// elements (no shell), so there is no injection risk from interface names or
// user-supplied addresses.
//
// UNVERIFIED: not run against a real admin shell yet. Watch the interface
// name with a space (e.g. "Ethernet 2"): Go quotes the whole argv element as
// "name=Ethernet 2" on the Windows command line, and netsh's own parser may
// expect the quotes around the value ("Ethernet 2"). If set fails with an
// interface-not-found error, that's the cause.
type Configurator struct{}

func (Configurator) SetStatic(cfg core.StaticConfig) error {
	ifi, _, err := activeInterface()
	if err != nil {
		return err
	}

	nameArg := "name=" + ifi.Name
	args := []string{"interface", "ip", "set", "address", nameArg, "static", cfg.IP.String(), cfg.Mask.String()}
	if cfg.Gateway != nil {
		args = append(args, cfg.Gateway.String())
	}
	if out, err := exec.Command("netsh", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("netsh set address: %w: %s", err, out)
	}

	for i, dns := range cfg.DNS {
		var dnsArgs []string
		if i == 0 {
			dnsArgs = []string{"interface", "ip", "set", "dns", nameArg, "static", dns.String()}
		} else {
			dnsArgs = []string{"interface", "ip", "add", "dns", nameArg, "addr=" + dns.String(), fmt.Sprintf("index=%d", i+1)}
		}
		if out, err := exec.Command("netsh", dnsArgs...).CombinedOutput(); err != nil {
			return fmt.Errorf("netsh set dns: %w: %s", err, out)
		}
	}
	return nil
}

func (Configurator) SetDHCP() error {
	ifi, _, err := activeInterface()
	if err != nil {
		return err
	}
	nameArg := "name=" + ifi.Name

	args := []string{"interface", "ip", "set", "address", nameArg, "dhcp"}
	if out, err := exec.Command("netsh", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("netsh set dhcp: %w: %s", err, out)
	}
	dnsArgs := []string{"interface", "ip", "set", "dns", nameArg, "dhcp"}
	if out, err := exec.Command("netsh", dnsArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("netsh set dns dhcp: %w: %s", err, out)
	}
	return nil
}
