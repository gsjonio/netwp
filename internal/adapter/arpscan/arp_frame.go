package arpscan

import (
	"encoding/binary"
	"net"
)

// Ethernet+ARP frame layout shared by the Linux raw-socket scanner. Kept
// build-tag-free (pure byte manipulation, no syscalls) so it can be unit
// tested from any OS.
const (
	etherTypeARP = 0x0806
	arpHTypeEth  = 1
	arpPTypeIPv4 = 0x0800
	arpOpRequest = 1
	arpOpReply   = 2
	arpFrameLen  = 14 + 28 // Ethernet header + ARP payload
)

var broadcastMAC = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

// buildARPRequest assembles a broadcast Ethernet+ARP "who has dstIP" frame.
func buildARPRequest(srcMAC net.HardwareAddr, srcIP, dstIP net.IP) []byte {
	frame := make([]byte, arpFrameLen)

	copy(frame[0:6], broadcastMAC)
	copy(frame[6:12], srcMAC)
	binary.BigEndian.PutUint16(frame[12:14], etherTypeARP)

	arp := frame[14:]
	binary.BigEndian.PutUint16(arp[0:2], arpHTypeEth)
	binary.BigEndian.PutUint16(arp[2:4], arpPTypeIPv4)
	arp[4] = 6 // hardware address length
	arp[5] = 4 // protocol address length
	binary.BigEndian.PutUint16(arp[6:8], arpOpRequest)
	copy(arp[8:14], srcMAC)
	copy(arp[14:18], srcIP.To4())
	// arp[18:24] (target hardware address) left zeroed: unknown, that's what we're asking.
	copy(arp[24:28], dstIP.To4())

	return frame
}

// parseARPReply extracts the sender's IP/MAC from a frame, if it is an ARP
// reply addressed to ourIP.
func parseARPReply(frame []byte, ourIP net.IP) (senderIP net.IP, senderMAC net.HardwareAddr, ok bool) {
	if len(frame) < arpFrameLen {
		return nil, nil, false
	}
	if binary.BigEndian.Uint16(frame[12:14]) != etherTypeARP {
		return nil, nil, false
	}
	arp := frame[14:]
	if binary.BigEndian.Uint16(arp[6:8]) != arpOpReply {
		return nil, nil, false
	}
	targetIP := net.IP(arp[24:28])
	if !targetIP.Equal(ourIP.To4()) {
		return nil, nil, false
	}
	senderIP = append(net.IP{}, arp[14:18]...)
	senderMAC = append(net.HardwareAddr{}, arp[8:14]...)
	return senderIP, senderMAC, true
}

// htons converts a 16-bit value to network byte order, needed for the
// AF_PACKET protocol field on Linux.
func htons(v uint16) uint16 {
	return (v<<8)&0xff00 | v>>8
}
