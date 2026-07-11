package core

import "net"

// MagicPacket builds a Wake-on-LAN magic packet for mac: 6 bytes of 0xFF
// followed by the target's 6-byte MAC repeated 16 times (the de-facto WoL
// format). Returns nil for a MAC that isn't 6 bytes, since WoL is defined only
// for EUI-48 addresses.
func MagicPacket(mac net.HardwareAddr) []byte {
	if len(mac) != 6 {
		return nil
	}
	packet := make([]byte, 0, 6+16*len(mac))
	for i := 0; i < 6; i++ {
		packet = append(packet, 0xFF)
	}
	for i := 0; i < 16; i++ {
		packet = append(packet, mac...)
	}
	return packet
}
