//go:build linux

package netinfo

import (
	"bufio"
	"encoding/hex"
	"net"
	"os"
	"strings"
)

// DefaultGateway reads the kernel routing table for the default route.
//
// ponytail: parses /proc/net/route directly instead of using netlink
// sockets. Good enough for a single default-route lookup; multiple routing
// tables (VRF, policy routing) aren't handled.
func DefaultGateway() net.IP {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan() // header line
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 || fields[1] != "00000000" { // Destination
			continue
		}
		raw, err := hex.DecodeString(fields[2]) // Gateway, little-endian hex
		if err != nil || len(raw) != 4 {
			continue
		}
		return net.IPv4(raw[3], raw[2], raw[1], raw[0])
	}
	return nil
}
