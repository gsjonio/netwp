//go:build windows

// Package icmpping measures ICMP round-trip time.
//
// Windows implementation: the IP Helper IcmpSendEcho API, which sends a real
// ICMP echo with no administrator rights and no raw socket (unlike a raw
// AF_INET ICMP socket, which Windows restricts). Same iphlpapi.dll the ARP
// scanner already uses.
package icmpping

import (
	"encoding/binary"
	"net"
	"syscall"
	"time"
	"unsafe"
)

var (
	iphlp          = syscall.NewLazyDLL("iphlpapi.dll")
	procIcmpCreate = iphlp.NewProc("IcmpCreateFile")
	procIcmpSend   = iphlp.NewProc("IcmpSendEcho")
	procIcmpClose  = iphlp.NewProc("IcmpCloseHandle")
	invalidHandle  = ^uintptr(0)
)

// Pinger implements core.Pinger.
type Pinger struct{}

func New() Pinger { return Pinger{} }

// icmpTTLOffset is where IP_OPTION_INFORMATION.Ttl lands inside
// ICMP_ECHO_REPLY on 64-bit Windows: Address(4) + Status(4) + RoundTripTime(4)
// + DataSize(2) + Reserved(2) + Data pointer(8, 64-bit) = 24, then Ttl is the
// option struct's first byte.
//
// Verified live: the machine running netwp (Windows) reported TTL 128, its
// router (embedded Linux) reported 64, matching their real OS families.
const icmpTTLOffset = 24

// Ping sends one ICMP echo to ip and returns the round-trip time and TTL.
func (Pinger) Ping(ip net.IP, timeout time.Duration) (time.Duration, int, bool) {
	ip4 := ip.To4()
	if ip4 == nil {
		return 0, 0, false
	}
	handle, _, _ := procIcmpCreate.Call()
	if handle == 0 || handle == invalidHandle {
		return 0, 0, false
	}
	defer procIcmpClose.Call(handle) //nolint:errcheck // best-effort cleanup

	req := []byte("netwp-icmp-echo-request-padding!") // 32 bytes of payload
	// Reply buffer: ICMP_ECHO_REPLY header + our payload + 8 bytes of slack, as
	// the API requires room for the reply struct plus the echoed data.
	reply := make([]byte, 128)
	dest := uintptr(binary.LittleEndian.Uint32(ip4)) // IPAddr, network byte order

	ms := uint32(timeout / time.Millisecond)
	if ms == 0 {
		ms = 1000
	}

	n, _, _ := procIcmpSend.Call(
		handle,
		dest,
		uintptr(unsafe.Pointer(&req[0])),
		uintptr(len(req)),
		0, // no IP options
		uintptr(unsafe.Pointer(&reply[0])),
		uintptr(len(reply)),
		uintptr(ms),
	)
	if n == 0 {
		return 0, 0, false
	}
	// ICMP_ECHO_REPLY: Address[0:4], Status[4:8], RoundTripTime[8:12] (all ULONG).
	if status := binary.LittleEndian.Uint32(reply[4:8]); status != 0 {
		return 0, 0, false // not IP_SUCCESS
	}
	rttMs := binary.LittleEndian.Uint32(reply[8:12])
	ttl := int(reply[icmpTTLOffset])
	return time.Duration(rttMs) * time.Millisecond, ttl, true
}
