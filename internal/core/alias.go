package core

import "net"

// Alias is a user-defined nickname bound to a device's MAC address. Keying on
// the MAC (not the IP) keeps the label stable across DHCP lease changes.
type Alias struct {
	MAC  net.HardwareAddr
	Name string
}

// ClassPin is a user-pinned device class bound to a MAC, overriding the
// automatic guess. Same MAC-keyed rationale as Alias.
type ClassPin struct {
	MAC   net.HardwareAddr
	Class DeviceClass
}
