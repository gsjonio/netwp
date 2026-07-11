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

// Prober performs a bounded concurrent TCP connect scan.
type Prober struct {
	Timeout time.Duration // per-port connect timeout
	Ports   []int         // ports to probe; nil uses the default probePorts set
}

func New() Prober { return Prober{Timeout: 300 * time.Millisecond} }

// OpenPorts returns the subset of the probed ports that accepted a connection.
func (p Prober) OpenPorts(ctx context.Context, ip net.IP) []int {
	ports := p.Ports
	if ports == nil {
		ports = probePorts
	}
	dialer := net.Dialer{Timeout: p.Timeout}
	host := ip.String()

	var (
		mu   sync.Mutex
		open []int
		wg   sync.WaitGroup
	)
	for _, port := range ports {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
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
