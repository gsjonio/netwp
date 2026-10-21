package main

import (
	"bytes"
	"net"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	"github.com/gsjonio/netwp/internal/adapter/eventlog"
	"github.com/gsjonio/netwp/internal/core"
)

func TestDoctorJSON(t *testing.T) {
	checks := []core.Check{
		{Name: "Gateway", OK: true, Detail: "responds"},
		{Name: "DNS", OK: false, Detail: "no resolve"},
	}
	got := doctorJSON(checks)
	if len(got) != 2 {
		t.Fatalf("got %d, want 2", len(got))
	}
	if got[0].Name != "Gateway" || !got[0].OK || got[0].Detail != "responds" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[1].OK {
		t.Errorf("got[1].OK = true, want false")
	}
}

func TestFilterByClass(t *testing.T) {
	devices := []core.Device{
		{IP: net.IPv4(192, 168, 1, 1), Class: core.ClassRouter},
		{IP: net.IPv4(192, 168, 1, 2), Class: core.ClassMedia},
		{IP: net.IPv4(192, 168, 1, 3), Class: core.ClassMedia},
	}
	got := filterByClass(devices, core.ClassMedia)
	if len(got) != 2 {
		t.Fatalf("filterByClass(media) = %d devices, want 2", len(got))
	}
	if filterByClass(devices, core.ClassPrinter) == nil {
		t.Error("filterByClass should return an empty non-nil slice, not nil")
	}
}

func TestListJSONMappers(t *testing.T) {
	mac1, _ := net.ParseMAC("aa:bb:cc:dd:ee:01")
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:02")

	// Empty stores must map to a non-nil slice, so --json emits [] not null.
	if got := aliasesJSON(nil); got == nil || len(got) != 0 {
		t.Errorf("aliasesJSON(nil) = %v, want non-nil empty", got)
	}
	if got := classesJSON(nil); got == nil || len(got) != 0 {
		t.Errorf("classesJSON(nil) = %v, want non-nil empty", got)
	}
	if got := watchedJSON(nil); got == nil || len(got) != 0 {
		t.Errorf("watchedJSON(nil) = %v, want non-nil empty", got)
	}

	// MACs serialize as their canonical string; class as its label.
	if got := aliasesJSON([]core.Alias{{MAC: mac1, Name: "PC"}}); got[0].MAC != "aa:bb:cc:dd:ee:01" || got[0].Name != "PC" {
		t.Errorf("aliasesJSON = %+v", got[0])
	}
	if got := classesJSON([]core.ClassPin{{MAC: mac1, Class: core.ClassMobile}}); got[0].MAC != "aa:bb:cc:dd:ee:01" || got[0].Class != "mobile" {
		t.Errorf("classesJSON = %+v", got[0])
	}
	if got := watchedJSON([]net.HardwareAddr{mac1, mac2}); len(got) != 2 || got[1].MAC != "aa:bb:cc:dd:ee:02" {
		t.Errorf("watchedJSON = %+v", got)
	}
}

func TestEventsForJSON(t *testing.T) {
	// A nil slice (missing log) must become an empty array, not null, so
	// `events --json` always emits valid JSON.
	if got := eventsForJSON(nil); got == nil || len(got) != 0 {
		t.Errorf("eventsForJSON(nil) = %v, want a non-nil empty slice", got)
	}
	in := []eventlog.Entry{{Kind: "joined", IP: "192.168.1.5"}}
	if got := eventsForJSON(in); len(got) != 1 || got[0].IP != "192.168.1.5" {
		t.Errorf("eventsForJSON passthrough = %v, want the one entry", got)
	}
}

func TestPlainEvent(t *testing.T) {
	mac, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	dev := func(alias string) core.Device {
		return core.Device{IP: net.IPv4(192, 168, 1, 7), MAC: mac, Alias: alias}
	}
	cases := []struct {
		kind    core.EventKind
		alias   string
		watched bool
		want    string // substring the line must contain
	}{
		{core.Joined, "", false, "joined  192.168.1.7 (192.168.1.7) (unknown)"},
		{core.Joined, "PC", false, "joined  PC (192.168.1.7)"},
		{core.Left, "PC", true, "left    PC (192.168.1.7) (watched)"},
		{core.Left, "PC", false, "left    PC (192.168.1.7)"},
	}
	for _, c := range cases {
		got := plainEvent(core.Event{Kind: c.kind, Device: dev(c.alias)}, c.watched)
		if !strings.Contains(got, c.want) {
			t.Errorf("plainEvent(%v, watched=%v) = %q, want it to contain %q", c.kind, c.watched, got, c.want)
		}
		if strings.Contains(got, "\x1b") {
			t.Errorf("plainEvent produced ANSI escapes: %q", got)
		}
	}
}

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

func TestPortSet(t *testing.T) {
	// An absent flag must stay nil so the prober keeps its curated default.
	got, err := portSet("")
	if err != nil || got != nil {
		t.Errorf("portSet(\"\") = %v, %v; want nil, nil", got, err)
	}
	got, err = portSet("22,443")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 22 || got[1] != 443 {
		t.Errorf("portSet = %v, want [22 443]", got)
	}
	if _, err := portSet("nope"); err == nil {
		t.Error("portSet(\"nope\"): expected an error, got nil")
	}
}

// TestPortsCmdRejectsBadPortSet guards the --ports wiring on `netwp ports`: an
// invalid list must fail while handling the flag, before any probe is dialed.
func TestPortsCmdRejectsBadPortSet(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"ports", "192.168.1.1", "--ports=bogus"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err == nil {
		t.Error("expected an error for an invalid --ports value")
	}
}

// TestMonitorCmdRejectsBadInterval guards the --interval wiring: a bad value
// must fail while handling the flag, before the monitor starts scanning.
func TestMonitorCmdRejectsBadInterval(t *testing.T) {
	for _, arg := range []string{"--interval=bogus", "--interval=10ms"} {
		root := newRootCmd()
		root.SetArgs([]string{"monitor", arg})
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err == nil {
			t.Errorf("monitor %s: expected an error, got nil", arg)
		}
	}
}

// TestScanCmdRejectsConflictingOutput guards the --csv flag groups: cobra must
// reject the combination up front, before a scan starts, rather than letting
// one output flag silently win over the other.
func TestScanCmdRejectsConflictingOutput(t *testing.T) {
	for _, args := range [][]string{
		{"scan", "--json", "--csv"},
		{"scan", "--diff", "--csv"},
	} {
		root := newRootCmd()
		root.SetArgs(args)
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		if err := root.Execute(); err == nil {
			t.Errorf("%v: expected an error for conflicting output flags, got nil", args)
		}
	}
}

func TestRootCmdHasCommands(t *testing.T) {
	root := newRootCmd()
	have := map[string]bool{}
	for _, c := range root.Commands() {
		have[c.Name()] = true
	}
	// The documented subcommands must all be registered (cobra adds `help` and
	// `completion` on its own).
	for _, want := range []string{"scan", "monitor", "dashboard", "speedtest", "iface", "alias", "class", "watch", "ports", "wake", "doctor", "events", "version", "update", "uninstall"} {
		if !have[want] {
			t.Errorf("root command is missing subcommand %q", want)
		}
	}
}

func TestRootCmdValidates(t *testing.T) {
	// A bad flag value should surface as an error through Execute, not panic.
	root := newRootCmd()
	root.SetArgs([]string{"scan", "--class=bogus"})
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	if err := root.Execute(); err == nil {
		t.Error("expected an error for an unknown --class value")
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

func TestParseInterval(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"1s", time.Second}, // exactly at the floor
		{"30s", 30 * time.Second},
		{"2m", 2 * time.Minute},
		{"1h30m", 90 * time.Minute},
	}
	for _, c := range cases {
		got, err := parseInterval(c.in)
		if err != nil {
			t.Errorf("parseInterval(%q): unexpected error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseInterval(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseIntervalInvalid(t *testing.T) {
	// Unparseable values, plus anything under the 1s floor: a sub-second
	// cadence would sweep the subnet back to back.
	for _, in := range []string{"", "30", "fast", "10ms", "999ms", "0s", "-5s"} {
		if _, err := parseInterval(in); err == nil {
			t.Errorf("parseInterval(%q): expected error, got nil", in)
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
