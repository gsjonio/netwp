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

// probePorts is a deliberately small set of ports whose presence hints at a
// device class (web UI, file sharing, printing, casting, phone sync).
//
// ponytail: keep this list short. A full port sweep would be slower and read as
// far more intrusive; these few are enough to classify common home devices.
var probePorts = []int{22, 80, 443, 445, 3389, 515, 631, 8009, 9100, 62078}

// Prober performs a bounded concurrent TCP connect scan.
type Prober struct {
	Timeout time.Duration // per-port connect timeout
}

func New() Prober { return Prober{Timeout: 300 * time.Millisecond} }

// OpenPorts returns the subset of probePorts that accepted a connection.
func (p Prober) OpenPorts(ctx context.Context, ip net.IP) []int {
	dialer := net.Dialer{Timeout: p.Timeout}
	host := ip.String()

	var (
		mu   sync.Mutex
		open []int
		wg   sync.WaitGroup
	)
	for _, port := range probePorts {
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
