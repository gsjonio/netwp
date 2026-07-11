package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/gsjonio/netwp/internal/core"
)

// bell is a tea.Cmd that rings the terminal (ASCII BEL) to audibly flag an
// alert event. Written to stderr so it never lands in bubbletea's stdout
// render buffer; the terminal still rings.
func bell() tea.Msg {
	fmt.Fprint(os.Stderr, "\a")
	return nil
}

const (
	logLimit           = 8                      // recent-activity lines kept on screen
	spinnerRate        = 120 * time.Millisecond // spinner frame cadence
	monitorSampleEvery = 1 * time.Second        // bandwidth sample cadence, when enabled
)

var (
	styOnline  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styOffline = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styHead    = lipgloss.NewStyle().Bold(true)
	styTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	styAlias   = lipgloss.NewStyle().Foreground(lipgloss.Color("51"))
	styWarn    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("203"))

	spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// MonitorConfig holds everything RunMonitor needs. Interval is the gap between
// scans; ScanBudget bounds how long each scan may run (kept separate: a scan
// can legitimately take longer than the interval).
//
// Reader + AlertDown enable an optional bandwidth alert: a nil Reader (or
// AlertDown <= 0) leaves monitor with no bandwidth line at all. Logger, if
// non-nil, persists join/leave events for `netwp events`. Watchlist, if
// non-nil, drives the "watched device left" alert (highlight + bell).
type MonitorConfig struct {
	Discovery  *core.Discovery
	Tracker    *core.Tracker
	Network    core.Network
	Interval   time.Duration
	ScanBudget time.Duration
	Reader     core.CounterReader // nil disables bandwidth sampling/alerting
	AlertDown  float64            // bytes/sec threshold; <= 0 disables the alert
	Logger     core.EventLogger   // nil disables event persistence
	Watchlist  core.Watchlist     // nil disables watched-device-left alerts
}

// RunMonitor starts the live monitoring UI and blocks until the user quits.
func RunMonitor(cfg MonitorConfig) error {
	m := monitorModel{
		discovery:  cfg.Discovery,
		tracker:    cfg.Tracker,
		network:    cfg.Network,
		interval:   cfg.Interval,
		scanBudget: cfg.ScanBudget,
		reader:     cfg.Reader,
		alertDown:  cfg.AlertDown,
		logger:     cfg.Logger,
		watchlist:  cfg.Watchlist,
		meter:      &core.RateMeter{},
		scanning:   true,
		scanStart:  time.Now(),
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

type monitorModel struct {
	discovery  *core.Discovery
	tracker    *core.Tracker
	network    core.Network
	interval   time.Duration
	scanBudget time.Duration
	reader     core.CounterReader // nil disables bandwidth sampling/alerting entirely
	alertDown  float64            // bytes/sec threshold; <= 0 disables the alert
	logger     core.EventLogger   // nil disables event persistence
	watchlist  core.Watchlist     // nil disables watched-device-left alerts
	meter      *core.RateMeter
	rate       core.Rate

	log       []string
	lastScan  time.Time
	scanning  bool
	scanStart time.Time
	frame     int
	err       error
	height    int
}

type scanDoneMsg struct {
	devices []core.Device
	at      time.Time
	err     error
}

type rescanMsg struct{}

type spinnerMsg struct{}

type monitorSampleMsg struct{ c core.NetCounters }

type monitorSampleTickMsg struct{}

func spinnerTick() tea.Cmd {
	return tea.Tick(spinnerRate, func(time.Time) tea.Msg { return spinnerMsg{} })
}

func (m monitorModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.scanNow, spinnerTick()}
	if m.reader != nil {
		cmds = append(cmds, m.readSample)
	}
	return tea.Batch(cmds...)
}

// scanNow runs one discovery pass off the UI goroutine (it is a tea.Cmd).
func (m monitorModel) scanNow() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), m.scanBudget)
	defer cancel()
	devices, err := m.discovery.Run(ctx, m.network)
	return scanDoneMsg{devices: devices, at: time.Now(), err: err}
}

// readSample is only ever called when m.reader != nil (the bandwidth alert
// is enabled) -- see Init and the monitorSampleTickMsg case in Update.
func (m monitorModel) readSample() tea.Msg {
	c, err := m.reader.Counters()
	if err != nil {
		return monitorSampleMsg{} // a failed read just yields a zero sample this tick
	}
	return monitorSampleMsg{c: c}
}

func (m monitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			if !m.scanning {
				m.scanning = true
				m.scanStart = time.Now()
				return m, m.scanNow
			}
		}

	case spinnerMsg:
		m.frame++
		return m, spinnerTick()

	case scanDoneMsg:
		m.scanning = false
		m.lastScan = msg.at
		m.err = msg.err
		alert := false
		if msg.err == nil {
			for _, e := range m.tracker.Observe(msg.devices, msg.at) {
				watched := m.watchlist != nil && m.watchlist.IsWatched(e.Device.MAC)
				m.log = append(m.log, formatEvent(e, watched))
				if m.logger != nil {
					_ = m.logger.Log(e)
				}
				if isAlertEvent(e, watched) {
					alert = true
				}
			}
			if len(m.log) > logLimit {
				m.log = m.log[len(m.log)-logLimit:]
			}
		}
		next := tea.Tick(m.interval, func(time.Time) tea.Msg { return rescanMsg{} })
		if alert {
			return m, tea.Batch(next, bell)
		}
		return m, next

	case rescanMsg:
		if !m.scanning {
			m.scanning = true
			m.scanStart = time.Now()
			return m, m.scanNow
		}

	case monitorSampleMsg:
		m.rate = m.meter.Update(msg.c, time.Now())
		return m, tick(monitorSampleEvery, monitorSampleTickMsg{})

	case monitorSampleTickMsg:
		return m, m.readSample
	}
	return m, nil
}

// bandwidthLine renders the current down/up rate, highlighted when the
// download rate has dropped below alertDown. Empty when bandwidth sampling
// is disabled (reader == nil).
func (m monitorModel) bandwidthLine() string {
	if m.reader == nil {
		return ""
	}
	line := fmt.Sprintf("↓ %s   ↑ %s", rateStr(m.rate.DownBps), rateStr(m.rate.UpBps))
	if m.alertDown > 0 && m.rate.DownBps < m.alertDown {
		return styWarn.Render(fmt.Sprintf("⚠ download below %s — %s", rateStr(m.alertDown), line))
	}
	return line
}

func (m monitorModel) View() string {
	devices := m.tracker.Devices()
	total := len(devices)
	online := 0
	for _, d := range devices {
		if d.Online {
			online++
		}
	}
	summary := fmt.Sprintf("%s  %s · %d online / %d known",
		styTitle.Render("netwp monitor"), m.network.CIDR, online, total)

	var activity string
	if len(m.log) > 0 {
		activity = styHead.Render("Recent activity") + "\n" + strings.Join(m.log, "\n")
	}

	var state string
	if m.scanning {
		spin := styTitle.Render(spinnerFrames[m.frame%len(spinnerFrames)])
		state = fmt.Sprintf("%s scanning… %.1fs", spin, time.Since(m.scanStart).Seconds())
	} else {
		state = styOffline.Render(fmt.Sprintf("idle · next in %s", time.Until(m.lastScan.Add(m.interval)).Round(time.Second)))
	}
	footer := state + styOffline.Render("   ·   r rescan   ·   q quit")
	bwLine := m.bandwidthLine()

	// Trim the device table to whatever vertical room is left, so the
	// footer never scrolls off screen on a short terminal. Fixed overhead:
	// summary line + blank, the table's own border/header/separator/border
	// (4, independent of row count), the blank line after it, the activity
	// block (if any) plus its trailing blank, an error line (if any), the
	// bandwidth line (if any) plus its trailing blank, and the footer itself.
	if m.height > 0 {
		used := lineCount(summary) + 1 + 4 + 1 + lineCount(footer)
		if activity != "" {
			used += lineCount(activity) + 1
		}
		if bwLine != "" {
			used += lineCount(bwLine) + 1
		}
		if m.err != nil {
			used++
		}
		var truncated bool
		budget := m.height - used
		devices, truncated = truncateToHeight(devices, budget)
		if truncated {
			summary += fmt.Sprintf(" (showing %d)", budget)
		}
	}

	var b strings.Builder
	b.WriteString(summary + "\n\n")
	b.WriteString(renderMonitorTable(devices, m.lastScan))
	b.WriteString("\n\n")
	if activity != "" {
		b.WriteString(activity + "\n\n")
	}
	if bwLine != "" {
		b.WriteString(bwLine + "\n\n")
	}
	if m.err != nil {
		b.WriteString(styOffline.Render("error: "+m.err.Error()) + "\n")
	}
	b.WriteString(footer)
	return b.String()
}

// truncateToHeight caps devices to budget rows, so a caller with a known
// terminal height never renders a table taller than it can show. budget<=0
// leaves devices unchanged: either the height isn't known yet, or the fixed
// chrome around the table already exceeds it, and showing everything is as
// reasonable a fallback as showing nothing.
func truncateToHeight(devices []core.TrackedDevice, budget int) (shown []core.TrackedDevice, truncated bool) {
	if budget <= 0 || len(devices) <= budget {
		return devices, false
	}
	return devices[:budget], true
}

func renderMonitorTable(devices []core.TrackedDevice, ref time.Time) string {
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(styOffline).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return styHead.Padding(0, 1)
			}
			if !devices[row].Online {
				return styOffline.Padding(0, 1)
			}
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers("", "IP", "ALIAS", "RTT", "TTL", "CLASS", "MAC", "HOSTNAME", "VENDOR", "PORTS", "LAST SEEN")

	for _, d := range devices {
		dot := styOffline.Render("○")
		if d.Online {
			dot = styOnline.Render("●")
		}
		t.Row(dot, d.IP.String(), aliasText(d.Alias), rttCellText(d.RTT, d.Reachable), TTLText(d.TTL), classLabel(d.Class), macText(d.MAC), orDash(d.Hostname), vendorText(d.Vendor), portsCellText(d.Ports), lastSeen(d, ref))
	}
	return t.String()
}

// isAlertEvent reports whether an event deserves a visual highlight and a
// terminal bell: an unrecognized device joining, or a watched device leaving.
func isAlertEvent(e core.Event, watched bool) bool {
	return (e.Kind == core.Joined && e.Device.Alias == "") || (e.Kind == core.Left && watched)
}

func formatEvent(e core.Event, watched bool) string {
	name := e.Device.Alias
	if name == "" {
		name = e.Device.Hostname
	}
	if name == "" {
		name = e.Device.IP.String()
	}
	ts := e.At.Format("15:04:05")
	switch {
	case e.Kind == core.Joined && e.Device.Alias == "":
		// A join with no alias is an unrecognized device: worth flagging.
		return styWarn.Render("⚠ " + ts + "  " + name + " joined (unknown)")
	case e.Kind == core.Joined:
		return styOnline.Render("＋") + " " + ts + "  " + name + " joined"
	case e.Kind == core.Left && watched:
		// A device the user asked to watch just dropped off: flag it.
		return styWarn.Render("⚠ " + ts + "  " + name + " left (watched)")
	default:
		return styOffline.Render("－") + " " + ts + "  " + name + " left"
	}
}

// aliasText renders a highlighted nickname, or the placeholder when unset.
func aliasText(alias string) string {
	if alias == "" {
		return dash
	}
	return styAlias.Render(alias)
}

// rttCellText colors round-trip time by quality tier, mirroring rttCell's
// thresholds (green good, bold neutral medium, red bad).
func rttCellText(rtt time.Duration, reachable bool) string {
	text := rttText(rtt, reachable)
	switch rttQualityOf(rtt, reachable) {
	case rttGood:
		return styOnline.Render(text)
	case rttMedium:
		return styHead.Render(text)
	case rttBad:
		return styWarn.Render(text)
	default:
		return text
	}
}

// portsCellText renders the PORTS cell, highlighted when a sensitive port
// (SSH, SMB, RDP) is open, so it catches the eye against the rest of the row.
func portsCellText(ports []int) string {
	text := portsText(ports)
	if hasSensitivePort(ports) {
		return styWarn.Render(text)
	}
	return text
}

func lastSeen(d core.TrackedDevice, ref time.Time) string {
	if d.Online {
		return "now"
	}
	secs := int(ref.Sub(d.LastSeen).Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds ago", secs)
	}
	return fmt.Sprintf("%dm ago", secs/60)
}
