// Package tcpprobe implements core.Prober with a light TCP connect scan of a
// few well-known ports. Pure stdlib net, so it is cross-platform.
package tcpprobe

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"
)

// probePorts is a curated set of ports common on home devices: enough to both
// hint at a device class (file sharing, printing, casting, phone sync) and show
// the user what a device actually exposes.
//
// ponytail: curated, not a full sweep. A 1-65535 scan would be far slower and
// read as an intrusion; this stays to well-known home-network services. Grow it
// only with ports that are both common and meaningful to a home user.
var probePorts = []int{
	21, 22, 23, 53, 80, 139, 443, 445, 515, 548, 554, 631,
	1883, 3000, 3306, 3389, 5000, 5432, 5900,
	8009, 8080, 8096, 8123, 8443, 8888, 9000, 9100, 32400, 62078,
}

// maxConcurrentDials caps how many sockets one OpenPorts call opens at once.
// Without it, ~29 ports times discovery's 32-device fan-out peaks near 900
// sockets, which can exhaust the file-descriptor limit (Linux default 1024) on
// a busy /24. 16 keeps the whole scan's peak around 512, with headroom.
const maxConcurrentDials = 16

// Prober performs a bounded concurrent TCP connect scan.
type Prober struct {
	Timeout time.Duration // per-port connect timeout
	Ports   []int         // ports to probe; nil uses the default probePorts set

	// dial defaults to a net.Dialer with Timeout; overridable in tests to
	// observe the concurrency cap without opening real sockets.
	dial func(ctx context.Context, network, addr string) (net.Conn, error)
}

func New() Prober { return Prober{Timeout: 300 * time.Millisecond} }

// OpenPorts returns the subset of the probed ports that accepted a connection.
func (p Prober) OpenPorts(ctx context.Context, ip net.IP) []int {
	ports := p.Ports
	if ports == nil {
		ports = probePorts
	}
	dial := p.dial
	if dial == nil {
		dial = (&net.Dialer{Timeout: p.Timeout}).DialContext
	}
	host := ip.String()

	var (
		mu   sync.Mutex
		open []int
		wg   sync.WaitGroup
	)
	sem := make(chan struct{}, maxConcurrentDials)
	for _, port := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			conn, err := dial(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
			if err != nil {
				return
			}
			conn.Close() //nolint:errcheck // just probing whether the port accepts a connection
			mu.Lock()
			open = append(open, port)
			mu.Unlock()
		}(port)
	}
	wg.Wait()
	return open
}
