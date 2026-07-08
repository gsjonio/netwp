//go:build !windows

package netinfo

// ponytail: Linux path not implemented yet. Real version would read
// /etc/resolv.conf for DNS servers and compare the address's PrefixOrigin via
// netlink (or `ip -j addr show`) for DHCP vs static.
func dnsServers(ifaceName string) []string { return nil }

func dhcpEnabled(ifaceName string) bool { return false }
