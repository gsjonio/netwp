//go:build linux

package netinfo

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// dnsServers reads nameserver entries from /etc/resolv.conf.
//
// ponytail: resolv.conf reflects the system resolver regardless of which
// interface is active; good enough since this tool only ever inspects one
// (the active) interface at a time.
func dnsServers(ifaceName string) []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close() //nolint:errcheck // read-only fd, best-effort cleanup

	var servers []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "nameserver" {
			servers = append(servers, fields[1])
		}
	}
	return servers
}

// dhcpEnabled reports whether systemd-networkd holds a DHCP lease for the
// interface.
//
// ponytail: only checks systemd-networkd's lease directory. Interfaces
// managed by NetworkManager, dhclient or netplan-without-networkd read as
// static; good enough for a best-effort inspect command. Widen this if it
// turns out to matter on the distros you actually run.
func dhcpEnabled(ifaceName string) bool {
	ifi, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join("/run/systemd/netif/leases", strconv.Itoa(ifi.Index)))
	return err == nil
}
