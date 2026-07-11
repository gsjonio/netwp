package tcpprobe

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestOpenPortsBoundsConcurrency proves the dial fan-out is capped: 100 ports
// would open 100 sockets at once without the semaphore. A fake dial records the
// peak number of overlapping calls and asserts it never exceeds the cap.
func TestOpenPortsBoundsConcurrency(t *testing.T) {
	var cur, peak int64
	var mu sync.Mutex
	ports := make([]int, 100)
	for i := range ports {
		ports[i] = i + 1
	}
	p := Prober{
		Ports: ports,
		dial: func(context.Context, string, string) (net.Conn, error) {
			c := atomic.AddInt64(&cur, 1)
			mu.Lock()
			if c > peak {
				peak = c
			}
			mu.Unlock()
			time.Sleep(2 * time.Millisecond) // hold the "socket" so calls overlap
			atomic.AddInt64(&cur, -1)
			return nil, errors.New("probe: closed") // treated as a closed port
		},
	}

	p.OpenPorts(context.Background(), net.ParseIP("127.0.0.1"))

	if peak > maxConcurrentDials {
		t.Errorf("peak concurrent dials = %d, want <= %d", peak, maxConcurrentDials)
	}
	if peak < 2 {
		t.Errorf("peak = %d, expected the dials to actually overlap", peak)
	}
}

// TestOpenPortsFindsListeningPort points the prober at a live loopback
// listener and a port that was bound then closed, and checks it reports only
// the one still accepting connections.
func TestOpenPortsFindsListeningPort(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	openPort := ln.Addr().(*net.TCPAddr).Port

	// Bind another ephemeral port and immediately release it, so it's a
	// port nothing is listening on for the duration of the probe.
	ln2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	closedPort := ln2.Addr().(*net.TCPAddr).Port
	ln2.Close()

	p := Prober{Timeout: 500 * time.Millisecond, Ports: []int{openPort, closedPort}}
	got := p.OpenPorts(context.Background(), net.ParseIP("127.0.0.1"))

	if len(got) != 1 || got[0] != openPort {
		t.Errorf("OpenPorts = %v, want [%d] (only the listening port)", got, openPort)
	}
}

// TestOpenPortsNoneOpen probes a port with nothing behind it and expects an
// empty result, not a nil-deref or a false positive.
func TestOpenPortsNoneOpen(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	p := Prober{Timeout: 300 * time.Millisecond, Ports: []int{port}}
	if got := p.OpenPorts(context.Background(), net.ParseIP("127.0.0.1")); len(got) != 0 {
		t.Errorf("OpenPorts = %v, want empty", got)
	}
}
