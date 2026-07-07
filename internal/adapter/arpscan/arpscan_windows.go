//go:build windows

// Package arpscan discovers hosts via active ARP.
//
// Windows implementation: the IP Helper SendARP API. SendARP issues a real ARP
// request when the address is not cached (active discovery, as required) yet
// needs no administrator rights and no Npcap/WinPcap driver.
//
// ponytail: SendARP covers the Windows-first goal with zero dependencies. If you
// later need raw-frame control (custom timing, non-local targets, passive
// sniffing) swap in a gopacket+Npcap adapter behind the same core.Scanner port.
package arpscan

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"syscall"
	"unsafe"

	"github.com/gsjonio/netwp/internal/core"
)

var sendARP = syscall.NewLazyDLL("iphlpapi.dll").NewProc("SendARP")

// Scanner probes every host in a subnet concurrently using SendARP.
type Scanner struct {
	Workers int // Bounded concurrency; <=0 uses a default.
}

func New() *Scanner { return &Scanner{Workers: 64} }

// Scan implements core.Scanner.
func (s *Scanner) Scan(ctx context.Context, target core.Network) ([]core.Device, error) {
	workers := s.Workers
	if workers <= 0 {
		workers = 64
	}

	jobs := make(chan net.IP)
	results := make(chan core.Device)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				mac, ok := resolveARP(ip)
				if !ok {
					continue
				}
				select {
				case results <- core.Device{IP: ip, MAC: mac, Online: true}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, ip := range target.Hosts() {
			select {
			case jobs <- ip:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() { wg.Wait(); close(results) }()

	var devices []core.Device
	for d := range results {
		devices = append(devices, d)
	}
	return devices, ctx.Err()
}

// resolveARP asks the stack for ip's MAC. Returns false if the host is silent.
//
// SendARP blocks with its own internal timeout, so a cancelled ctx stops new
// dispatches but will not interrupt a probe already in flight.
func resolveARP(ip net.IP) (net.HardwareAddr, bool) {
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, false
	}
	// IPAddr is a ULONG in network byte order; reading the 4 raw bytes as a
	// little-endian uint32 on x86 reproduces that layout.
	dest := binary.LittleEndian.Uint32(ip4)

	var mac [6]byte
	length := uint32(len(mac))
	ret, _, _ := sendARP.Call(
		uintptr(dest),
		0, // SrcIP 0: let the stack choose the source
		uintptr(unsafe.Pointer(&mac[0])),
		uintptr(unsafe.Pointer(&length)),
	)
	if ret != 0 || length == 0 {
		return nil, false
	}
	return net.HardwareAddr(mac[:length]), true
}
