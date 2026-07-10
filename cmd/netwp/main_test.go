package main

import (
	"bytes"
	"net"
	"runtime/debug"
	"strings"
	"testing"
)

func TestPrintUsageListsCommands(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()
	for _, want := range []string{"scan", "monitor", "dashboard", "speedtest", "iface", "alias", "ports", "version", "help"} {
		if !strings.Contains(out, want) {
			t.Errorf("usage output missing command %q:\n%s", want, out)
		}
	}
}

func TestVcsSetting(t *testing.T) {
	info := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "abc123def456"},
		{Key: "vcs.modified", Value: "true"},
	}}
	if got := vcsSetting(info, "vcs.revision"); got != "abc123def456" {
		t.Errorf("vcs.revision = %q", got)
	}
	if got := vcsSetting(info, "vcs.modified"); got != "true" {
		t.Errorf("vcs.modified = %q", got)
	}
	if got := vcsSetting(info, "missing"); got != "" {
		t.Errorf("missing key = %q, want empty", got)
	}
}

func TestPrintVersionRuns(t *testing.T) {
	// Exercises whichever of the two paths applies to the running test
	// binary's own build info (Go auto-embeds a pseudo-version from VCS
	// tags even for a plain `go build`/`go test`, not just `go install
	// pkg@vX`, so which branch actually runs depends on the checkout).
	var buf bytes.Buffer
	printVersion(&buf)
	if !strings.HasPrefix(buf.String(), "netwp ") {
		t.Errorf("printVersion() = %q, want it to start with \"netwp \"", buf.String())
	}
}

func TestParseStaticArgs(t *testing.T) {
	cfg, err := parseStaticArgs([]string{"192.168.1.50/24", "192.168.1.1", "8.8.8.8", "1.1.1.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IP.Equal(net.ParseIP("192.168.1.50")) {
		t.Errorf("IP = %v, want 192.168.1.50", cfg.IP)
	}
	if cfg.Mask.String() != "255.255.255.0" {
		t.Errorf("Mask = %v, want 255.255.255.0", cfg.Mask)
	}
	if !cfg.Gateway.Equal(net.ParseIP("192.168.1.1")) {
		t.Errorf("Gateway = %v, want 192.168.1.1", cfg.Gateway)
	}
	if len(cfg.DNS) != 2 || !cfg.DNS[0].Equal(net.ParseIP("8.8.8.8")) || !cfg.DNS[1].Equal(net.ParseIP("1.1.1.1")) {
		t.Errorf("DNS = %v, want [8.8.8.8 1.1.1.1]", cfg.DNS)
	}
}

func TestParseStaticArgsNoDNS(t *testing.T) {
	cfg, err := parseStaticArgs([]string{"10.0.0.5/8", "10.0.0.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.DNS) != 0 {
		t.Errorf("DNS = %v, want none", cfg.DNS)
	}
}

func TestParseStaticArgsInvalid(t *testing.T) {
	cases := [][]string{
		{},
		{"192.168.1.50/24"},              // missing gateway
		{"not-a-cidr", "192.168.1.1"},    // bad address
		{"192.168.1.50/24", "not-an-ip"}, // bad gateway
		{"192.168.1.50/24", "192.168.1.1", "not-dns"}, // bad DNS
	}
	for _, args := range cases {
		if _, err := parseStaticArgs(args); err == nil {
			t.Errorf("parseStaticArgs(%v): expected error, got nil", args)
		}
	}
}
