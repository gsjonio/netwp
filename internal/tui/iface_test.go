package tui

import (
	"bytes"
	"net"
	"strings"
	"testing"

	"github.com/gsjonio/netwp/internal/core"
)

func TestRenderInterface(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("192.168.1.42/24")
	info := core.InterfaceInfo{
		Name:       "Ethernet",
		MAC:        net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		IP:         net.ParseIP("192.168.1.42").To4(),
		CIDR:       cidr,
		Gateway:    net.ParseIP("192.168.1.1"),
		DNSServers: []net.IP{net.ParseIP("8.8.8.8"), net.ParseIP("1.1.1.1")},
		DHCP:       true,
	}

	var buf bytes.Buffer
	RenderInterface(&buf, info)
	out := buf.String()

	for _, want := range []string{
		"Ethernet", "192.168.1.42", "255.255.255.0", "dhcp",
		"192.168.1.1", "8.8.8.8, 1.1.1.1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q, got:\n%s", want, out)
		}
	}
}

func TestRenderInterfaceMissingGatewayAndDNS(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("10.0.0.5/8")
	info := core.InterfaceInfo{Name: "eth0", IP: net.ParseIP("10.0.0.5").To4(), CIDR: cidr}

	var buf bytes.Buffer
	RenderInterface(&buf, info)
	out := buf.String()

	if !strings.Contains(out, "static") {
		t.Errorf("expected static mode when DHCP is false, got:\n%s", out)
	}
	if !strings.Contains(out, "—") {
		t.Errorf("expected placeholder dash for missing gateway/dns, got:\n%s", out)
	}
}
