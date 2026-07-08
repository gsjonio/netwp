//go:build windows

package netinfo

import (
	"fmt"
	"os/exec"
	"strings"
)

// psQuote escapes an argument for embedding in a PowerShell single-quoted
// string, so an unusual adapter name can't break out into other commands.
func psQuote(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// dnsServers returns the configured IPv4 DNS servers for the named interface.
//
// ponytail: shells out to PowerShell instead of parsing IP_ADAPTER_ADDRESSES
// via raw syscalls. Simpler, and this is a read-only inspect command.
func dnsServers(ifaceName string) []string {
	cmd := fmt.Sprintf(`(Get-DnsClientServerAddress -InterfaceAlias '%s' -AddressFamily IPv4).ServerAddresses`, psQuote(ifaceName))
	out, err := exec.Command("powershell", "-NoProfile", "-Command", cmd).Output()
	if err != nil {
		return nil
	}
	var servers []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			servers = append(servers, line)
		}
	}
	return servers
}

// dhcpEnabled reports whether the interface's IPv4 address was assigned by DHCP.
func dhcpEnabled(ifaceName string) bool {
	cmd := fmt.Sprintf(`(Get-NetIPAddress -InterfaceAlias '%s' -AddressFamily IPv4).PrefixOrigin`, psQuote(ifaceName))
	out, err := exec.Command("powershell", "-NoProfile", "-Command", cmd).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Dhcp")
}
