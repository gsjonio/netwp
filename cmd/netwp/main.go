// Command netwp is a terminal network manager. This entry point is the
// composition root: the cobra command tree lives in root.go, and the run*
// handlers live in sibling files by area (scan.go, iface.go, live.go, cmds.go);
// wire.go holds the adapter wiring and MAC resolution shared across them.
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
	// cobra owns dispatch, flags, and per-command help (see root.go). Errors are
	// printed here, in netwp's "netwp: <err>" style, instead of cobra's default
	// "Error: ..." + usage dump (SilenceErrors/SilenceUsage on the root).
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "netwp:", err)
		os.Exit(1)
	}
}

// version names the release when set at build time via
// -ldflags "-X main.version=vX.Y.Z" (release.yml does this on a tag). It wins
// over the build-info lookup below, so a downloaded release binary reports its
// tag instead of the "(devel)" a plain `go build` embeds.
var version string

// versionString reports the release set via ldflags, else the version embedded
// by `go install module@vX.Y.Z`, else the VCS commit `go build` embeds
// automatically (Go 1.18+) when built from a local source tree.
func versionString() string {
	if version != "" {
		return "netwp " + version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "netwp (unknown version)"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return "netwp " + v
	}

	rev := vcsSetting(info, "vcs.revision")
	if rev == "" {
		return "netwp (devel)"
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	dirty := ""
	if vcsSetting(info, "vcs.modified") == "true" {
		dirty = "-dirty"
	}
	return fmt.Sprintf("netwp (devel, commit %s%s)", rev, dirty)
}

// printVersion writes the version line for the `version` command.
func printVersion(w io.Writer) {
	fmt.Fprintln(w, versionString())
}

func vcsSetting(info *debug.BuildInfo, key string) string {
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}
