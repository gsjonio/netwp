package arpscan

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestBuildARPRequestLayout(t *testing.T) {
	srcMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	srcIP := net.ParseIP("192.168.1.10")
	dstIP := net.ParseIP("192.168.1.20")

	frame := buildARPRequest(srcMAC, srcIP, dstIP)
	if len(frame) != arpFrameLen {
		t.Fatalf("frame length = %d, want %d", len(frame), arpFrameLen)
	}
	if got := net.HardwareAddr(frame[0:6]).String(); got != broadcastMAC.String() {
		t.Errorf("dst MAC = %s, want broadcast", got)
	}
	if got := net.HardwareAddr(frame[6:12]).String(); got != srcMAC.String() {
		t.Errorf("src MAC = %s, want %s", got, srcMAC)
	}
	if got := binary.BigEndian.Uint16(frame[12:14]); got != etherTypeARP {
		t.Errorf("ethertype = %#x, want %#x", got, etherTypeARP)
	}
}

func TestARPRequestReplyRoundTrip(t *testing.T) {
	ourIP := net.ParseIP("192.168.1.10")
	peerMAC := net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	peerIP := net.ParseIP("192.168.1.20")

	// Simulate the peer's reply: same frame shape, opcode flipped, roles
	// swapped (peer is now the sender, we are the target).
	reply := buildARPRequest(peerMAC, peerIP, ourIP)
	binary.BigEndian.PutUint16(reply[14+6:14+8], arpOpReply)

	gotIP, gotMAC, ok := parseARPReply(reply, ourIP)
	if !ok {
		t.Fatal("expected parseARPReply to accept a matching reply")
	}
	if !gotIP.Equal(peerIP) {
		t.Errorf("sender IP = %v, want %v", gotIP, peerIP)
	}
	if gotMAC.String() != peerMAC.String() {
		t.Errorf("sender MAC = %v, want %v", gotMAC, peerMAC)
	}
}

func TestParseARPReplyRejectsRequests(t *testing.T) {
	frame := buildARPRequest(net.HardwareAddr{0, 1, 2, 3, 4, 5}, net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2"))
	if _, _, ok := parseARPReply(frame, net.ParseIP("10.0.0.2")); ok {
		t.Error("expected an ARP request (opcode 1) to be rejected, not treated as a reply")
	}
}

func TestParseARPReplyRejectsWrongTarget(t *testing.T) {
	frame := buildARPRequest(net.HardwareAddr{0, 1, 2, 3, 4, 5}, net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.2"))
	binary.BigEndian.PutUint16(frame[14+6:14+8], arpOpReply)

	if _, _, ok := parseARPReply(frame, net.ParseIP("10.0.0.99")); ok {
		t.Error("expected a reply addressed to a different IP to be rejected")
	}
}

func TestHtons(t *testing.T) {
	if got := htons(0x0806); got != 0x0608 {
		t.Errorf("htons(0x0806) = %#x, want 0x0608", got)
	}
}
