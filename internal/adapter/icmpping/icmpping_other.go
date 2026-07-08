//go:build !windows

package icmpping

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"net"
)

// Pinger implements core.Pinger by shelling out to the system `ping`.
//
// ponytail: an unprivileged ICMP datagram socket (SOCK_DGRAM/IPPROTO_ICMP,
// gated by net.ipv4.ping_group_range) would avoid the process spawn, but `ping`
// works everywhere without root and keeps this small. Upgrade if per-host spawn
// cost matters.
type Pinger struct{}

func New() Pinger { return Pinger{} }

func (Pinger) Ping(ip net.IP, timeout time.Duration) (time.Duration, bool) {
	secs := int(timeout / time.Second)
	if secs < 1 {
		secs = 1
	}
	out, err := exec.Command("ping", "-c", "1", "-W", strconv.Itoa(secs), ip.String()).Output()
	if err != nil {
		return 0, false
	}
	// Parse "time=1.23 ms" from the reply line.
	s := string(out)
	i := strings.Index(s, "time=")
	if i < 0 {
		return 0, false
	}
	rest := s[i+len("time="):]
	end := strings.IndexAny(rest, " \t")
	if end < 0 {
		return 0, false
	}
	ms, err := strconv.ParseFloat(rest[:end], 64)
	if err != nil {
		return 0, false
	}
	return time.Duration(ms * float64(time.Millisecond)), true
}
