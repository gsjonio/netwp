// Command netwp is a terminal network manager. This entry point is the
// composition root: main dispatches subcommands, and the run* handlers live in
// sibling files by area (scan.go, iface.go, live.go, cmds.go); wire.go holds
// the adapter wiring and MAC resolution shared across them.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"
)

// portNames labels the ports tcpprobe checks, for `netwp ports` output.
var portNames = map[int]string{
	21:    "FTP",
	22:    "SSH",
	23:    "Telnet",
	53:    "DNS",
	80:    "HTTP",
	139:   "NetBIOS",
	443:   "HTTPS",
	445:   "SMB",
	515:   "LPD (printing)",
	548:   "AFP (Apple file sharing)",
	554:   "RTSP (camera)",
	631:   "IPP (printing)",
	1883:  "MQTT (smart home)",
	3000:  "HTTP (app/dev)",
	3306:  "MySQL",
	3389:  "RDP",
	5000:  "UPnP / app",
	5432:  "PostgreSQL",
	5900:  "VNC",
	8009:  "Chromecast",
	8080:  "HTTP (alt)",
	8096:  "Jellyfin (media)",
	8123:  "Home Assistant",
	8443:  "HTTPS (alt)",
	8888:  "HTTP (alt)",
	9000:  "app / Portainer",
	9100:  "JetDirect (printing)",
	32400: "Plex (media)",
	62078: "iOS sync (lockdownd)",
}

func portName(p int) string {
	if name, ok := portNames[p]; ok {
		return name
	}
	return "unknown"
}

const (
	scanTimeout       = 20 * time.Second // one-shot scan budget
	monitorEvery      = 10 * time.Second // interval between monitor scans
	monitorScanBudget = 30 * time.Second // max time a single monitor scan may run
	offlineAfter      = 30 * time.Second // grace before a missing device is offline
	speedtestTimeout  = 30 * time.Second // download + upload budget
)

func main() {
	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	var err error
	switch command {
	case "", "help", "-h", "--help":
		printUsage(os.Stdout)
		return
	case "version", "--version":
		printVersion(os.Stdout)
		return
	case "update":
		err = runUpdate()
	case "scan", "--json", "--diff": // --json/--diff are scan flags, not their own subcommands
		err = runScan(hasArg("--json"), hasArg("--diff"))
	case "monitor":
		err = runMonitor()
	case "speedtest":
		err = runSpeedtest()
	case "iface":
		err = runIface()
	case "alias":
		err = runAlias()
	case "dashboard":
		err = runDashboard()
	case "ports":
		err = runPorts()
	case "wake":
		err = runWake()
	case "doctor":
		err = runDoctor()
	case "events":
		err = runEvents()
	case "class":
		err = runClass()
	case "watch":
		err = runWatch()
	case "uninstall":
		err = runUninstall()
	default:
		fmt.Fprintf(os.Stderr, "netwp: unknown command %q\n\n", command)
		printUsage(os.Stderr)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "netwp:", err)
		os.Exit(1)
	}
}

// printUsage lists every subcommand with a one-line description, in the
// style of standard CLIs (git, docker): running netwp with no arguments (or
// help/-h/--help) shows this instead of taking any action.
func printUsage(w io.Writer) {
	fmt.Fprint(w, `netwp - terminal network manager (ARP scan, monitor, dashboard, bandwidth, interface config)

Usage:
  netwp <command> [arguments]

Commands:
  scan [--json] [--diff] [--ports=<list>]         one-shot ARP scan of the local network, with per-device RTT
                                                   --diff prints only what changed since the last scan
                                                   --ports=22,80,443 probes a custom TCP port set instead of the default
  monitor [--quiet]                               live TUI: devices joining/leaving in real time (q to quit)
                                                   --quiet runs headless (no UI): one line per event to stdout, for a service/logfile
  dashboard                                       full dashboard: wifi + live bandwidth + speedtest + devices
  speedtest [--json]                              download/upload throughput test
  iface                                           inspect the active interface's IP config
  iface static <ip>/<bits> <gateway> [dns...]     set a static IP (asks to confirm)
  iface dhcp                                      switch back to DHCP (asks to confirm)
  alias set <ip-or-mac> <name>                    nickname a device
  alias ls                                        list nicknames
  alias rm <ip-or-mac>                            remove a nickname
  class set <ip-or-mac> <class>                   pin a device's class (router|computer|mobile|media|printer|iot)
  class ls                                        list class overrides
  class rm <ip-or-mac>                            remove a class override
  watch add <ip-or-mac>                           alert when this device leaves (monitor/dashboard)
  watch ls                                        list watched devices
  watch rm <ip-or-mac>                            stop watching a device
  ports <ip> [--json]                             open ports + RTT for one device
  wake <ip-or-mac-or-alias>                       send a Wake-on-LAN magic packet to power on a device
  doctor [--json]                                 diagnose connectivity (interface, gateway, internet, DNS, Wi-Fi)
  events [n] [--device=<alias-or-mac>]            show the last n join/leave events (default 20; --device filters to one)
  version                                         show the installed version
  update                                          update to the latest version (needs the Go toolchain)
  uninstall                                       remove netwp's local data (asks to confirm)
  help                                            show this help

Run "netwp scan" to see the devices on your network.
`)
}

// printVersion reports the version embedded by `go install module@vX.Y.Z`,
// or falls back to the VCS commit `go build` embeds automatically (Go
// 1.18+) when built from a local source tree instead of a tagged module.
func printVersion(w io.Writer) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Fprintln(w, "netwp (unknown version)")
		return
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		fmt.Fprintf(w, "netwp %s\n", v)
		return
	}

	rev := vcsSetting(info, "vcs.revision")
	if rev == "" {
		fmt.Fprintln(w, "netwp (devel)")
		return
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	dirty := ""
	if vcsSetting(info, "vcs.modified") == "true" {
		dirty = "-dirty"
	}
	fmt.Fprintf(w, "netwp (devel, commit %s%s)\n", rev, dirty)
}

func vcsSetting(info *debug.BuildInfo, key string) string {
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

// hasArg reports whether flag appears anywhere in the CLI arguments. The
// single place that interprets a flag, so subcommand funcs take their
// options as plain parameters instead of reaching back into os.Args.
func hasArg(flag string) bool {
	for _, a := range os.Args[1:] {
		if a == flag {
			return true
		}
	}
	return false
}
