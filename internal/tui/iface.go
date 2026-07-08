package tui

import (
	"fmt"
	"io"
	"net"

	"github.com/gsjonio/netwp/internal/core"
)

// RenderInterface prints the active interface's IP configuration.
func RenderInterface(w io.Writer, info core.InterfaceInfo) {
	mask := net.IP(info.CIDR.Mask).String()

	mode := "static"
	if info.DHCP {
		mode = "dhcp"
	}

	fmt.Fprintf(w, "%sinterface:%s %s\n", colorBold, colorReset, info.Name)
	fmt.Fprintf(w, "%smac:      %s %s\n", colorBold, colorReset, macCell(info.MAC).text)
	fmt.Fprintf(w, "%sip:       %s %s\n", colorBold, colorReset, info.IP)
	fmt.Fprintf(w, "%smask:     %s %s\n", colorBold, colorReset, mask)
	fmt.Fprintf(w, "%smode:     %s %s\n", colorBold, colorReset, mode)
	fmt.Fprintf(w, "%sgateway:  %s %s\n", colorBold, colorReset, textCell(ipString(info.Gateway)).text)
	fmt.Fprintf(w, "%sdns:      %s %s\n", colorBold, colorReset, textCell(dnsString(info.DNSServers)).text)
}

func ipString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

func dnsString(servers []net.IP) string {
	if len(servers) == 0 {
		return ""
	}
	s := servers[0].String()
	for _, ip := range servers[1:] {
		s += ", " + ip.String()
	}
	return s
}
