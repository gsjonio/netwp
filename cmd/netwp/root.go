package main

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/gsjonio/netwp/internal/core"
	"github.com/gsjonio/netwp/internal/tui"
)

// newRootCmd builds the full netwp command tree. cobra provides per-command
// --help and a native `completion` command; the root silences cobra's own error
// and usage output so main can print errors in netwp's "netwp: <err>" style.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "netwp",
		Short:         "terminal network manager (ARP scan, monitor, dashboard, bandwidth, interface config)",
		Long:          "netwp - terminal network manager.\n\nActive local-network device discovery (ARP), live monitoring, a full dashboard,\nbandwidth testing, and interface configuration. Run \"netwp scan\" to see the\ndevices on your network.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       versionString(),
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.AddCommand(
		newScanCmd(),
		newMonitorCmd(),
		newDashboardCmd(),
		newSpeedtestCmd(),
		newIfaceCmd(),
		newAliasCmd(),
		newClassCmd(),
		newWatchCmd(),
		newPortsCmd(),
		newWakeCmd(),
		newDoctorCmd(),
		newEventsCmd(),
		newVersionCmd(),
		newUpdateCmd(),
		newUninstallCmd(),
	)
	return root
}

func newScanCmd() *cobra.Command {
	var asJSON, diff bool
	var portsStr, classStr, sortStr string
	c := &cobra.Command{
		Use:   "scan",
		Short: "one-shot ARP scan of the local network, with per-device RTT",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			var ports []int
			if portsStr != "" {
				p, err := parsePorts(portsStr)
				if err != nil {
					return err
				}
				ports = p
			}
			var class core.DeviceClass
			filtered := false
			if classStr != "" {
				cl, ok := core.ParseClass(classStr)
				if !ok {
					return fmt.Errorf("unknown class %q: expected one of router, computer, mobile, media, printer, iot", classStr)
				}
				class, filtered = cl, true
			}
			sortBy, ok := tui.ParseSortColumn(sortStr)
			if sortStr != "" && !ok {
				return fmt.Errorf("unknown sort column %q: expected one of ip, rtt, name, class", sortStr)
			}
			return runScan(asJSON, diff, ports, class, filtered, sortBy)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON output")
	c.Flags().BoolVar(&diff, "diff", false, "print only what changed since the last scan")
	c.Flags().StringVar(&portsStr, "ports", "", "probe a custom TCP port set, e.g. 22,80,443")
	c.Flags().StringVar(&classStr, "class", "", "show only devices of one class (router|computer|mobile|media|printer|iot)")
	c.Flags().StringVar(&sortStr, "sort", "", "order the output by column (ip|rtt|name|class); default ip")
	return c
}

func newMonitorCmd() *cobra.Command {
	var quiet bool
	var alertStr string
	c := &cobra.Command{
		Use:   "monitor",
		Short: "live TUI: devices joining/leaving in real time (q to quit)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			var alertDown float64
			if alertStr != "" {
				r, err := parseRate(alertStr)
				if err != nil {
					return err
				}
				alertDown = r
			}
			return runMonitor(quiet, alertDown)
		},
	}
	c.Flags().BoolVar(&quiet, "quiet", false, "run headless (no UI): one line per event to stdout, for a service/logfile")
	c.Flags().StringVar(&alertStr, "alert-down", "", "flag a download rate drop, e.g. 50Mbps")
	return c
}

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "full dashboard: wifi + live bandwidth + speedtest + devices",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { return runDashboard() },
	}
}

func newSpeedtestCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "speedtest",
		Short: "download/upload throughput test",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { return runSpeedtest(asJSON) },
	}
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON output")
	return c
}

func newIfaceCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "iface",
		Short: "inspect the active interface's IP config",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { return runIfaceInspect() },
	}
	c.AddCommand(
		&cobra.Command{
			Use:   "static <ip>/<bits> <gateway> [dns...]",
			Short: "set a static IP (asks to confirm)",
			Args:  cobra.MinimumNArgs(2),
			RunE:  func(_ *cobra.Command, args []string) error { return runIfaceSetStatic(args) },
		},
		&cobra.Command{
			Use:   "dhcp",
			Short: "switch back to DHCP (asks to confirm)",
			Args:  cobra.NoArgs,
			RunE:  func(_ *cobra.Command, _ []string) error { return runIfaceSetDHCP() },
		},
	)
	return c
}

// newListCmd builds the `ls` subcommand shared by alias/class/watch: same shape
// (ls/list alias, no args, a --json flag), differing only in the description and
// the list function it delegates to.
func newListCmd(short string, list func(asJSON bool) error) *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:     "ls",
		Aliases: []string{"list"},
		Short:   short,
		Args:    cobra.NoArgs,
		RunE:    func(_ *cobra.Command, _ []string) error { return list(asJSON) },
	}
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON output")
	return c
}

func newAliasCmd() *cobra.Command {
	c := &cobra.Command{Use: "alias", Short: "nickname a device"}
	c.AddCommand(
		&cobra.Command{
			Use:   "set <ip-or-mac> <name>",
			Short: "nickname a device",
			Args:  cobra.MinimumNArgs(2),
			RunE:  func(_ *cobra.Command, args []string) error { return aliasSet(args[0], args[1:]) },
		},
		newListCmd("list nicknames", aliasList),
		&cobra.Command{
			Use:     "rm <ip-or-mac>",
			Aliases: []string{"remove", "del"},
			Short:   "remove a nickname",
			Args:    cobra.ExactArgs(1),
			RunE:    func(_ *cobra.Command, args []string) error { return aliasRemove(args[0]) },
		},
	)
	return c
}

func newClassCmd() *cobra.Command {
	c := &cobra.Command{Use: "class", Short: "pin a device's class (router|computer|mobile|media|printer|iot)"}
	c.AddCommand(
		&cobra.Command{
			Use:   "set <ip-or-mac> <class>",
			Short: "pin a device's class",
			Args:  cobra.ExactArgs(2),
			RunE:  func(_ *cobra.Command, args []string) error { return classSet(args[0], args[1]) },
		},
		newListCmd("list class overrides", classList),
		&cobra.Command{
			Use:     "rm <ip-or-mac>",
			Aliases: []string{"remove", "del"},
			Short:   "remove a class override",
			Args:    cobra.ExactArgs(1),
			RunE:    func(_ *cobra.Command, args []string) error { return classRemove(args[0]) },
		},
	)
	return c
}

func newWatchCmd() *cobra.Command {
	c := &cobra.Command{Use: "watch", Short: "alert when a device leaves (monitor/dashboard)"}
	c.AddCommand(
		&cobra.Command{
			Use:   "add <ip-or-mac>",
			Short: "watch a device",
			Args:  cobra.ExactArgs(1),
			RunE:  func(_ *cobra.Command, args []string) error { return watchAdd(args[0]) },
		},
		newListCmd("list watched devices", watchList),
		&cobra.Command{
			Use:     "rm <ip-or-mac>",
			Aliases: []string{"remove", "del"},
			Short:   "stop watching a device",
			Args:    cobra.ExactArgs(1),
			RunE:    func(_ *cobra.Command, args []string) error { return watchRemove(args[0]) },
		},
	)
	return c
}

func newPortsCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "ports <ip>",
		Short: "open ports + RTT for one device",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ip := net.ParseIP(args[0])
			if ip == nil {
				return fmt.Errorf("invalid IP %q", args[0])
			}
			return runPorts(ip, asJSON)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON output")
	return c
}

func newWakeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "wake <ip-or-mac-or-alias>",
		Short: "send a Wake-on-LAN magic packet to power on a device",
		Args:  cobra.ExactArgs(1),
		RunE:  func(_ *cobra.Command, args []string) error { return runWake(args[0]) },
	}
}

func newDoctorCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{
		Use:   "doctor",
		Short: "diagnose connectivity (interface, gateway, internet, DNS, Wi-Fi)",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { return runDoctor(asJSON) },
	}
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON output")
	return c
}

func newEventsCmd() *cobra.Command {
	var device string
	var asJSON bool
	c := &cobra.Command{
		Use:   "events [n]",
		Short: "show the last n join/leave events (default 20)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			n := 20
			if len(args) == 1 {
				v, err := strconv.Atoi(args[0])
				if err != nil || v <= 0 {
					return fmt.Errorf("invalid count %q: expected a positive number", args[0])
				}
				n = v
			}
			return runEvents(n, device, asJSON)
		},
	}
	c.Flags().StringVar(&device, "device", "", "filter to one device (alias or MAC)")
	c.Flags().BoolVar(&asJSON, "json", false, "machine-readable JSON output")
	return c
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "show the installed version",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { printVersion(os.Stdout); return nil },
	}
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "update to the latest version (needs the Go toolchain)",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { return runUpdate() },
	}
}

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "remove netwp's local data (asks to confirm)",
		Args:  cobra.NoArgs,
		RunE:  func(_ *cobra.Command, _ []string) error { return runUninstall() },
	}
}
