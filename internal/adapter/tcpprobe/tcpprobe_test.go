package tcpprobe

import (
	"context"
	"net"
	"testing"
	"time"
)

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
