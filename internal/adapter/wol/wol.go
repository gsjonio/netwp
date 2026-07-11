// Package wol implements core.Waker: it powers on a sleeping device by
// broadcasting a Wake-on-LAN magic packet. Pure stdlib net, cross-platform.
package wol

import (
	"fmt"
	"net"

	"github.com/gsjonio/netwp/internal/core"
)

// wolPort is UDP 9 (discard), the conventional Wake-on-LAN destination port.
const wolPort = 9

// Waker implements core.Waker.
type Waker struct{}

func New() Waker { return Waker{} }

// Wake broadcasts a magic packet for mac to the local network's limited
// broadcast address. The target NIC wakes if it was left with WoL enabled;
// there is no reply, so success here means "sent", not "woke".
func (Waker) Wake(mac net.HardwareAddr) error {
	packet := core.MagicPacket(mac)
	if packet == nil {
		return fmt.Errorf("wake-on-lan needs a 6-byte MAC, got %q", mac)
	}
	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: net.IPv4bcast, Port: wolPort})
	if err != nil {
		return fmt.Errorf("opening broadcast socket: %w", err)
	}
	defer conn.Close() //nolint:errcheck // best-effort cleanup
	if _, err := conn.Write(packet); err != nil {
		return fmt.Errorf("sending magic packet: %w", err)
	}
	return nil
}
