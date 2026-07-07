//go:build windows

package netinfo

import (
	"encoding/binary"
	"net"
	"syscall"
	"unsafe"
)

var getBestRoute = syscall.NewLazyDLL("iphlpapi.dll").NewProc("GetBestRoute")

// mibIPForwardRow mirrors MIB_IPFORWARDROW (14 DWORDs). We only read NextHop.
type mibIPForwardRow struct {
	Dest      uint32
	Mask      uint32
	Policy    uint32
	NextHop   uint32
	IfIndex   uint32
	Type      uint32
	Proto     uint32
	Age       uint32
	NextHopAS uint32
	Metric1   uint32
	Metric2   uint32
	Metric3   uint32
	Metric4   uint32
	Metric5   uint32
}

// DefaultGateway returns the IPv4 default gateway (the router), or nil.
//
// It asks the routing table for the best route to a public address; the route's
// next hop is the gateway. A zero next hop means the destination is on-link
// (no gateway), so we return nil.
func DefaultGateway() net.IP {
	const dest = 8<<0 | 8<<8 | 8<<16 | 8<<24 // 8.8.8.8 in network byte order
	var row mibIPForwardRow
	ret, _, _ := getBestRoute.Call(uintptr(dest), 0, uintptr(unsafe.Pointer(&row)))
	if ret != 0 || row.NextHop == 0 {
		return nil
	}
	ip := make(net.IP, 4)
	binary.LittleEndian.PutUint32(ip, row.NextHop)
	return ip
}
