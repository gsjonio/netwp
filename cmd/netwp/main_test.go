package main

import (
	"bytes"
	"net"
	"runtime/debug"
	"strings"
	"testing"
)

func TestParsePorts(t *testing.T) {
	got, err := parsePorts("22, 80,443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 || got[0] != 22 || got[1] != 80 || got[2] != 443 {
		t.Errorf("parsePorts = %v, want [22 80 443]", got)
	}
}

func TestParsePortsInvalid(t *testing.T) {
	for _, in := range []string{"", "80,abc", "0", "70000", "-1"} {
		if _, err := parsePorts(in); err == nil {
			t.Errorf("parsePorts(%q): expected error, got nil", in)
		}
	}
}

func TestPortsFlag(t *testing.T) {
	got, err := portsFlag([]string{"--json", "--ports=8080,3000"})
	if err != nil || len(got) != 2 || got[0] != 8080 || got[1] != 3000 {
		t.Errorf("portsFlag = (%v, %v), want [8080 3000]", got, err)
	}
	if got, _ := portsFlag([]string{"--json"}); got != nil {
		t.Errorf("portsFlag with no --ports = %v, want nil (default set)", got)
	}
}

func TestPrintUsageListsCommands(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()
	for _, want := range []string{"scan", "monitor", "dashboard", "speedtest", "iface", "alias", "ports", "version", "update", "help"} {
		if !strings.Contains(out, want) {
			t.Errorf("usage output missing command %q:\n%s", want, out)
		}
	}
}

// TestRunUpdateNoGoToolchain guards the fallback message when `go` isn't on
// PATH (e.g. a binary downloaded from Releases, no dev toolchain installed):
// it must fail with a clear pointer to the Releases page, not try to exec a
// missing binary and return a raw "file not found".
func TestRunUpdateNoGoToolchain(t *testing.T) {
	t.Setenv("PATH", "")
	err := runUpdate()
	if err == nil {
		t.Fatal("expected an error when go is not on PATH, got nil")
	}
	if !strings.Contains(err.Error(), "releases") {
		t.Errorf("error = %q, want it to point at the Releases page", err.Error())
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

func TestParseRate(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"50Mbps", 50e6 / 8},
		{"1.5Gbps", 1.5e9 / 8},
		{"200Kbps", 200e3 / 8},
		{"800bps", 100},
	}
	for _, c := range cases {
		got, err := parseRate(c.in)
		if err != nil {
			t.Errorf("parseRate(%q): unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseRate(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseRateInvalid(t *testing.T) {
	for _, in := range []string{"", "50", "fastMbps"} {
		if _, err := parseRate(in); err == nil {
			t.Errorf("parseRate(%q): expected error, got nil", in)
		}
	}
}

func TestParseAlertFlag(t *testing.T) {
	got, err := parseAlertFlag([]string{"--alert-down=50Mbps"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := 50e6 / 8; got != want {
		t.Errorf("parseAlertFlag = %v, want %v", got, want)
	}

	got, err = parseAlertFlag(nil)
	if err != nil || got != 0 {
		t.Errorf("parseAlertFlag(nil) = (%v, %v), want (0, nil)", got, err)
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
