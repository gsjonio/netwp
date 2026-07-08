//go:build linux

// Linux implementation: raw ARP requests over an AF_PACKET socket, since
// Linux has no admin-free ARP API like Windows' SendARP.
//
// ponytail: requires CAP_NET_RAW (root, or `setcap cap_net_raw+ep` on the
// binary). Written and cross-compiled (GOOS=linux) from a Windows dev
// machine; not yet run against real Linux hardware, so treat the socket
// plumbing here as unverified until it's exercised on a real box.
package arpscan

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/gsjonio/netwp/internal/core"
)

// collectWindow bounds how long Scan keeps reading replies after firing all
// probes. ARP replies on a LAN arrive in milliseconds, so a short window is
// plenty and keeps a fast network from waiting out the whole ctx budget.
//
// ponytail: fixed 2s. If replies trickle in on a large/slow segment, widen
// it or reset the timer on each reply (quiet-period detection).
const collectWindow = 2 * time.Second

// Scanner sends ARP requests over a raw socket bound to the outbound
// interface and collects replies until ctx is done.
//
// ponytail: assumes ctx carries a deadline, same as every caller in this
// project (scan/monitor always use context.WithTimeout). Without one, Scan
// blocks until the caller cancels.
type Scanner struct{}

func New() *Scanner { return &Scanner{} }

// Scan implements core.Scanner.
func (s *Scanner) Scan(ctx context.Context, target core.Network) ([]core.Device, error) {
	ifi, err := outboundInterface(target)
	if err != nil {
		return nil, err
	}

	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(etherTypeARP)))
	if err != nil {
		return nil, fmt.Errorf("open raw socket (needs CAP_NET_RAW/root): %w", err)
	}
	defer syscall.Close(fd)

	addr := &syscall.SockaddrLinklayer{Protocol: htons(etherTypeARP), Ifindex: ifi.Index}
	if err := syscall.Bind(fd, addr); err != nil {
		return nil, fmt.Errorf("bind to %s: %w", ifi.Name, err)
	}
	// Short receive timeout so the reader loop notices ctx cancellation
	// instead of blocking forever once the network goes quiet.
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &syscall.Timeval{Usec: 200_000})

	srcIP := target.Self.To4()

	var mu sync.Mutex
	found := make(map[string]core.Device)
	readerDone := make(chan struct{})

	// The reader stops on whichever comes first: the caller's deadline, or a
	// short collection window after all probes are sent. Without this cap a
	// fast LAN would still block for the whole ctx budget (and the monitor,
	// which passes a 30s budget, would never keep its interval).
	scanCtx, cancel := context.WithTimeout(ctx, collectWindow)
	defer cancel()

	go func() {
		defer close(readerDone)
		buf := make([]byte, 128)
		for {
			if scanCtx.Err() != nil {
				return
			}
			n, _, err := syscall.Recvfrom(fd, buf, 0)
			if err != nil {
				continue // recv timeout or transient error; loop and recheck ctx
			}
			if ip, mac, ok := parseARPReply(buf[:n], srcIP); ok {
				mu.Lock()
				found[ip.String()] = core.Device{IP: ip, MAC: mac, Online: true}
				mu.Unlock()
			}
		}
	}()

	for _, ip := range target.Hosts() {
		if scanCtx.Err() != nil {
			break
		}
		frame := buildARPRequest(ifi.HardwareAddr, srcIP, ip)
		_ = syscall.Sendto(fd, frame, 0, addr)
	}

	<-scanCtx.Done()
	<-readerDone

	mu.Lock()
	defer mu.Unlock()
	devices := make([]core.Device, 0, len(found))
	for _, d := range found {
		devices = append(devices, d)
	}
	// Best-effort, same contract as the Windows scanner: return whatever
	// answered instead of discarding a partial scan.
	return devices, nil
}

// outboundInterface finds the interface carrying target.Self, the address
// whose subnet is being scanned.
func outboundInterface(target core.Network) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for i := range ifaces {
		addrs, err := ifaces[i].Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.Equal(target.Self) {
				return &ifaces[i], nil
			}
		}
	}
	return nil, fmt.Errorf("no interface with address %s", target.Self)
}
