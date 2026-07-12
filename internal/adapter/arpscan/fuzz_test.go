package arpscan

import (
	"net"
	"testing"
)

// FuzzParseARPReply throws arbitrary bytes at the ARP frame parser: reply
// frames arrive off a raw socket from untrusted hosts on the segment, so a
// malformed one must never panic. Run the full fuzzer with:
//
//	go test ./internal/adapter/arpscan -run=x -fuzz=FuzzParseARPReply
func FuzzParseARPReply(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 42)) // 14-byte Ethernet header + 28-byte ARP, all zero
	ourIP := net.IPv4(192, 168, 1, 10)
	f.Fuzz(func(t *testing.T, frame []byte) {
		_, _, _ = parseARPReply(frame, ourIP) // must not panic
	})
}
