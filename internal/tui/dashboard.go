package tui

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gsjonio/netwp/internal/core"
)

// Refresh cadences. Bandwidth samples fast; the netsh/scan/speedtest work is
// slower and spaced out so the dashboard stays responsive.
const (
	dashSampleEvery = 1 * time.Second
	dashWifiEvery   = 6 * time.Second
	dashScanEvery   = 15 * time.Second
	dashSpeedEvery  = 5 * time.Minute
	dashPingEvery   = 5 * time.Second
	dashScanBudget  = 30 * time.Second
	dashSpeedBudget = 40 * time.Second
	dashPingTarget  = "8.8.8.8" // internet-latency probe target
	dashDefaultCols = 112
	dashNarrowCols  = 90 // below this, the three top panels stack instead of sitting side by side
	histLen         = 24
)

// DashboardConfig holds everything RunDashboard needs. Logger, if non-nil,
// persists join/leave events for `netwp events`; Watchlist, if non-nil, drives
// the "watched device left" alert (highlight + bell).
type DashboardConfig struct {
	Discovery *core.Discovery
	Tracker   *core.Tracker
	Network   core.Network
	Info      core.InterfaceInfo
	Reader    core.CounterReader
	WiFi      core.WiFiInspector
	Speed     *core.Speedtest
	Pinger    core.Pinger
	Logger    core.EventLogger // nil disables event persistence
	Watchlist core.Watchlist   // nil disables watched-device-left alerts
}

// RunDashboard starts the composite live dashboard and blocks until the user
// quits.
func RunDashboard(cfg DashboardConfig) error {
	m := dashModel{
		discovery: cfg.Discovery,
		tracker:   cfg.Tracker,
		network:   cfg.Network,
		info:      cfg.Info,
		reader:    cfg.Reader,
		wifi:      cfg.WiFi,
		speed:     cfg.Speed,
		pinger:    cfg.Pinger,
		logger:    cfg.Logger,
		watchlist: cfg.Watchlist,
		meter:     &core.RateMeter{},
		start:     time.Now(),
		width:     dashDefaultCols,
		ops:       []string{opLine("dashboard started")},
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

type dashModel struct {
	discovery *core.Discovery
	tracker   *core.Tracker
	network   core.Network
	info      core.InterfaceInfo
	reader    core.CounterReader
	wifi      core.WiFiInspector
	speed     *core.Speedtest
	pinger    core.Pinger
	logger    core.EventLogger // nil disables event persistence
	watchlist core.Watchlist   // nil disables watched-device-left alerts
	meter     *core.RateMeter

	start      time.Time
	rate       core.Rate
	downHist   []float64
	wifiInfo   core.WiFiInfo
	wifiErr    error
	wifiHist   []float64
	result     core.BandwidthResult
	speedAt    time.Time
	speedErr   error
	speedHist  []float64
	netLatency time.Duration
	netUp      bool
	netHist    []float64
	lastScan   time.Time
	log        []string // device join/leave events (ACTIVITY panel)
	ops        []string // operation log: scans, speedtests, state changes (LOG panel)
	width      int
	height     int

	filter    string // active DEVICES-table filter query
	filtering bool   // true while the user is typing the filter
	sort      SortKey
}

// opsLimit is how many operation-log lines the LOG panel keeps on screen.
const opsLimit = 8

// appendLog appends s to lines, trimmed to the last limit entries.
func appendLog(lines []string, s string, limit int) []string {
	lines = append(lines, s)
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines
}

// opLine prefixes an operation message with a dim timestamp for the LOG panel.
func opLine(msg string) string {
	return styOffline.Render(time.Now().Format("15:04:05")) + " " + msg
}

// lastN returns the last n elements of s (all of them if fewer, none if n<=0).
func lastN(s []string, n int) []string {
	if n <= 0 {
		return nil
	}
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// pushHist appends v to hist, trimmed to the last histLen samples.
func pushHist(hist []float64, v float64) []float64 {
	hist = append(hist, v)
	if len(hist) > histLen {
		hist = hist[len(hist)-histLen:]
	}
	return hist
}

type (
	sampleMsg     struct{ c core.NetCounters }
	sampleTickMsg struct{}
	wifiMsg       struct {
		info core.WiFiInfo
		err  error
	}
	wifiTickMsg struct{}
	scanMsg     struct {
		devices []core.Device
		at      time.Time
		err     error
	}
	scanTickMsg struct{}
	speedMsg    struct {
		result core.BandwidthResult
		at     time.Time
		err    error
	}
	speedTickMsg struct{}
	pingMsg      struct {
		rtt time.Duration
		ok  bool
	}
	pingTickMsg struct{}
)

func (m dashModel) readSample() tea.Msg {
	c, err := m.reader.Counters()
	if err != nil {
		return sampleMsg{} // a failed read just yields a zero sample this tick
	}
	return sampleMsg{c: c}
}

func (m dashModel) readWifi() tea.Msg {
	info, err := m.wifi.WiFi()
	return wifiMsg{info: info, err: err}
}

func (m dashModel) scan() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), dashScanBudget)
	defer cancel()
	devices, err := m.discovery.Run(ctx, m.network)
	return scanMsg{devices: devices, at: time.Now(), err: err}
}

func (m dashModel) runSpeed() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), dashSpeedBudget)
	defer cancel()
	r, err := m.speed.Run(ctx)
	return speedMsg{result: r, at: time.Now(), err: err}
}

func (m dashModel) readPing() tea.Msg {
	// TTL is a LAN-device fingerprinting hint; irrelevant for a fixed
	// internet target like 8.8.8.8, so it's discarded here.
	rtt, _, ok := m.pinger.Ping(net.ParseIP(dashPingTarget), 800*time.Millisecond)
	return pingMsg{rtt: rtt, ok: ok}
}

func tick(d time.Duration, msg tea.Msg) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return msg })
}

func (m dashModel) Init() tea.Cmd {
	return tea.Batch(m.readSample, m.readWifi, m.scan, m.runSpeed, m.readPing)
}

func (m dashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filtering {
			switch msg.Type {
			case tea.KeyCtrlC:
				return m, tea.Quit
			case tea.KeyEsc:
				m.filtering, m.filter = false, "" // esc discards the filter
			case tea.KeyEnter:
				m.filtering = false // enter keeps it applied
			case tea.KeyBackspace:
				m.filter = applyFilterKey(m.filter, nil, true)
			case tea.KeyRunes, tea.KeySpace:
				m.filter = applyFilterKey(m.filter, msg.Runes, false)
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.filter != "" {
				m.filter = "" // esc clears an applied filter before it quits
				return m, nil
			}
			return m, tea.Quit
		case "/":
			m.filtering = true
			return m, nil
		case "s":
			m.sort = m.sort.next()
			return m, nil
		case "r":
			m.ops = appendLog(m.ops, opLine("running scan (manual)…"), opsLimit)
			// A manual rescan re-resolves names instead of serving the cache,
			// so a stale hostname is what pressing r actually refreshes.
			m.discovery.ResetResolverCache()
			return m, m.scan
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case sampleMsg:
		m.rate = m.meter.Update(msg.c, time.Now())
		m.downHist = pushHist(m.downHist, m.rate.DownBps)
		return m, tick(dashSampleEvery, sampleTickMsg{})
	case sampleTickMsg:
		return m, m.readSample

	case wifiMsg:
		// Log only connection changes, not every 6s refresh, to keep the log
		// signal-heavy (a wired host never logs a wi-fi line at all).
		if msg.err == nil && (msg.info.Connected != m.wifiInfo.Connected || msg.info.SSID != m.wifiInfo.SSID) {
			if msg.info.Connected {
				m.ops = appendLog(m.ops, opLine(fmt.Sprintf("wi-fi: connected to %s (%d%%)", msg.info.SSID, msg.info.SignalPercent)), opsLimit)
			} else {
				m.ops = appendLog(m.ops, opLine("wi-fi: disconnected"), opsLimit)
			}
		}
		m.wifiInfo, m.wifiErr = msg.info, msg.err
		if msg.err == nil && msg.info.Connected {
			m.wifiHist = pushHist(m.wifiHist, float64(msg.info.SignalPercent))
		}
		return m, tick(dashWifiEvery, wifiTickMsg{})
	case wifiTickMsg:
		return m, m.readWifi

	case scanMsg:
		m.lastScan = msg.at
		if msg.err != nil {
			// A failed scan used to vanish silently (devices, _ := Run). Surface
			// it in the LOG panel and keep the last good device list on screen.
			m.ops = appendLog(m.ops, opLine("scan failed: "+msg.err.Error()), opsLimit)
			return m, tick(dashScanEvery, scanTickMsg{})
		}
		m.ops = appendLog(m.ops, opLine(fmt.Sprintf("scan done · %d devices", len(msg.devices))), opsLimit)
		alert := false
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
		if alert {
			return m, tea.Batch(tick(dashScanEvery, scanTickMsg{}), bell)
		}
		return m, tick(dashScanEvery, scanTickMsg{})
	case scanTickMsg:
		m.ops = appendLog(m.ops, opLine("running scan…"), opsLimit)
		return m, m.scan

	case speedMsg:
		m.result, m.speedAt, m.speedErr = msg.result, msg.at, msg.err
		if msg.err != nil {
			m.ops = appendLog(m.ops, opLine("speedtest failed: "+msg.err.Error()), opsLimit)
		} else {
			m.speedHist = pushHist(m.speedHist, msg.result.DownloadMbps)
			m.ops = appendLog(m.ops, opLine(fmt.Sprintf("speedtest · %.0f↓ / %.0f↑ Mbps", msg.result.DownloadMbps, msg.result.UploadMbps)), opsLimit)
		}
		return m, tick(dashSpeedEvery, speedTickMsg{})
	case speedTickMsg:
		m.ops = appendLog(m.ops, opLine("running speedtest…"), opsLimit)
		return m, m.runSpeed

	case pingMsg:
		// Log only up/down transitions, not every 5s probe.
		if msg.ok != m.netUp {
			if msg.ok {
				m.ops = appendLog(m.ops, opLine(fmt.Sprintf("internet up (%s to %s)", msg.rtt.Round(time.Millisecond), dashPingTarget)), opsLimit)
			} else {
				m.ops = appendLog(m.ops, opLine("internet down (no reply from "+dashPingTarget+")"), opsLimit)
			}
		}
		m.netLatency, m.netUp = msg.rtt, msg.ok
		if msg.ok {
			m.netHist = pushHist(m.netHist, float64(msg.rtt.Microseconds())/1000)
		}
		return m, tick(dashPingEvery, pingTickMsg{})
	case pingTickMsg:
		return m, m.readPing
	}
	return m, nil
}

func (m dashModel) View() string {
	width := m.width
	if width <= 0 {
		width = dashDefaultCols // no WindowSizeMsg yet
	}

	header := m.renderHeader(width)
	var top string
	if width < dashNarrowCols {
		// Too narrow for three ~22-column panels side by side (they'd wrap
		// mid-word); stack them instead, each using the full width.
		top = lipgloss.JoinVertical(lipgloss.Left,
			panel("WI-FI", m.renderWifi(), width-2),
			panel("BANDWIDTH", m.renderBandwidth(), width-2),
			panel("SPEEDTEST", m.renderSpeed(), width-2),
		)
	} else {
		colW := (width - 4) / 3
		top = lipgloss.JoinHorizontal(lipgloss.Top,
			panel("WI-FI", m.renderWifi(), colW),
			panel("BANDWIDTH", m.renderBandwidth(), colW),
			panel("SPEEDTEST", m.renderSpeed(), colW),
		)
	}
	footer := styOffline.Render(fmt.Sprintf("/ filter · s sort: %s · r rescan · q quit", m.sort))

	var activity string
	if len(m.log) > 0 {
		activity = panel("ACTIVITY", strings.Join(m.log, "\n"), width-2)
	}

	allDevices := m.tracker.Devices()
	total := len(allDevices)
	online := 0
	for _, d := range allDevices {
		if d.Online {
			online++
		}
	}
	devices := filterDevices(allDevices, m.filter)
	sortDevices(devices, m.sort)
	devTitle := fmt.Sprintf("DEVICES · %d online / %d known", online, total)
	if m.filtering {
		devTitle += "  ·  filter: " + m.filter + "▌"
	} else if m.filter != "" {
		devTitle += fmt.Sprintf("  ·  filter %q (%d match)", m.filter, len(devices))
	}

	// LOG panel at the bottom: a running trace of the dashboard's own work
	// (scans, speedtests, connectivity changes), so you can see what it's doing.
	// It yields to the device table and footer on a short terminal -- shrinking,
	// then hiding -- so the footer never scrolls off. Fixed overhead below the
	// header/top/activity/footer lines: the device panel's own border (2) +
	// title (1), plus the inner table's border+header+separator+border (4).
	opsPanel := ""
	if m.height > 0 {
		fixed := lineCount(header) + lineCount(top) + lineCount(footer) + 7
		if activity != "" {
			fixed += lineCount(activity)
		}
		room := m.height - fixed // shared by the device rows and the LOG panel
		// Show the LOG only where there's slack, capped at opsLimit and at most
		// half the slack, so the device table keeps the rest.
		if logLines := min(len(m.ops), opsLimit, (room-1)/2); logLines > 0 {
			opsPanel = panel("LOG", strings.Join(lastN(m.ops, logLines), "\n"), width-2)
			room -= lineCount(opsPanel)
		}
		var truncated bool
		devices, truncated = truncateToHeight(devices, room)
		if truncated {
			devTitle = fmt.Sprintf("DEVICES · %d online / %d known (showing %d)", online, total, room)
		}
	} else {
		opsPanel = panel("LOG", strings.Join(lastN(m.ops, opsLimit), "\n"), width-2)
	}
	if summary := classSummary(allDevices); summary != "" {
		devTitle += " · " + summary
	}
	devBody := renderMonitorTable(devices, m.lastScan)

	// The table sizes itself to its own content (more columns over time --
	// PORTS and TTL both landed after this panel's width was fixed at
	// width-2). If it now needs more room than that, let the panel grow to
	// fit instead of forcing lipgloss to wrap an already-bordered table:
	// that mangles the box-drawing characters, and doubles every row's line
	// count, which is worse than a panel occasionally wider than the other
	// three above it.
	devPanelWidth := width - 2
	if bw := lipgloss.Width(devBody); bw > devPanelWidth {
		devPanelWidth = bw
	}

	parts := []string{header, top}
	if activity != "" {
		parts = append(parts, activity)
	}
	parts = append(parts, panel(devTitle, devBody, devPanelWidth+2))
	if opsPanel != "" {
		parts = append(parts, opsPanel)
	}
	parts = append(parts, footer)
	return strings.Join(parts, "\n")
}

func lineCount(s string) int { return strings.Count(s, "\n") + 1 }

// classSummary returns a compact breakdown of online devices by class, e.g.
// "2 Router · 1 IoT", sorted by count descending. ClassUnknown and
// ClassThisDevice are skipped: neither tells you anything about what's on
// your network. Empty when nothing is left to summarize.
func classSummary(devices []core.TrackedDevice) string {
	counts := map[core.DeviceClass]int{}
	for _, d := range devices {
		if d.Online {
			counts[d.Class]++
		}
	}
	type entry struct {
		class core.DeviceClass
		n     int
	}
	var entries []entry
	for c, n := range counts {
		if c == core.ClassUnknown || c == core.ClassThisDevice {
			continue
		}
		entries = append(entries, entry{c, n})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].n != entries[j].n {
			return entries[i].n > entries[j].n
		}
		return entries[i].class < entries[j].class // stable order for equal counts
	})
	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = fmt.Sprintf("%d %s", e.n, e.class)
	}
	return strings.Join(parts, " · ")
}

func (m dashModel) renderHeader(width int) string {
	title := styTitle.Render("netwp dashboard")
	clock := time.Now().Format("15:04:05")
	netRange := ""
	if lo, hi, ok := sparklineRange(m.netHist); ok {
		netRange = fmt.Sprintf(" %.0f-%.0fms", lo, hi)
	}
	left := fmt.Sprintf("%s   %s · IP %s · GW %s · net %s %s%s · up %s",
		title, m.info.Name, m.info.IP, ipOr(m.info.Gateway, "?"),
		netStyle(m.netUp).Render(rttText(m.netLatency, m.netUp)), sparkline(m.netHist), netRange, uptime(m.start))
	gap := width - lipgloss.Width(left) - len(clock) - 2
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + styHead.Render(clock)
}

func (m dashModel) renderWifi() string {
	if m.wifiErr != nil {
		return styOffline.Render("no Wi-Fi interface")
	}
	w := m.wifiInfo
	if !w.Connected {
		s := styOffline.Render("disconnected")
		if n := len(w.Nearby); n > 0 {
			s += fmt.Sprintf("\n%d APs visible", n)
		}
		return s
	}
	sig := fmt.Sprintf("%d%% (%d dBm)", w.SignalPercent, w.SignalDBM())
	best := w.RecommendChannel()
	channelHint := styOffline.Render("clear")
	if best != w.Channel {
		channelHint = styHead.Render(fmt.Sprintf("try ch %d", best))
	}
	wifiRange := ""
	if lo, hi, ok := sparklineRange(m.wifiHist); ok {
		wifiRange = fmt.Sprintf(" %.0f-%.0f%%", lo, hi)
	}
	lines := []string{
		styAlias.Render(w.SSID),
		"signal   " + signalStyle(w.SignalPercent).Render(sig),
		"         " + sparkline(m.wifiHist) + wifiRange,
		fmt.Sprintf("channel  %d  %s  %s", w.Channel, w.Band, channelHint),
		fmt.Sprintf("rate     %d/%d Mbps", w.RxRateMbps, w.TxRateMbps),
		fmt.Sprintf("nearby   %d APs · %d on ch %d", len(w.Nearby), w.SameChannelCount(), w.Channel),
	}
	return strings.Join(lines, "\n")
}

func (m dashModel) renderBandwidth() string {
	bwLine := sparkline(m.downHist)
	if lo, hi, ok := sparklineRange(m.downHist); ok {
		bwLine += fmt.Sprintf(" %s-%s", rateStr(lo), rateStr(hi))
	}
	lines := []string{
		"down  " + styOnline.Render(rateStr(m.rate.DownBps)),
		"up    " + styHead.Render(rateStr(m.rate.UpBps)),
		bwLine,
		styOffline.Render(fmt.Sprintf("RX %s · TX %s", byteStr(m.rate.TotalRx), byteStr(m.rate.TotalTx))),
	}
	return strings.Join(lines, "\n")
}

func (m dashModel) renderSpeed() string {
	if m.speedAt.IsZero() {
		return styOffline.Render("testing…")
	}
	if m.speedErr != nil {
		return styOffline.Render("error: " + m.speedErr.Error())
	}
	next := time.Until(m.speedAt.Add(dashSpeedEvery)).Round(time.Second)
	speedLine := sparkline(m.speedHist)
	if lo, hi, ok := sparklineRange(m.speedHist); ok {
		speedLine += fmt.Sprintf(" %.1f-%.1f Mbps", lo, hi)
	}
	return strings.Join([]string{
		fmt.Sprintf("down  %s", styOnline.Render(fmt.Sprintf("%.1f Mbps", m.result.DownloadMbps))),
		fmt.Sprintf("up    %s", styHead.Render(fmt.Sprintf("%.1f Mbps", m.result.UploadMbps))),
		speedLine,
		styOffline.Render("at " + m.speedAt.Format("15:04:05")),
		styOffline.Render(fmt.Sprintf("next in %s", next)),
	}, "\n")
}

// panel wraps body in a titled rounded box of the given outer width.
func panel(title, body string, width int) string {
	inner := width - 2
	if inner < 8 {
		inner = 8
	}
	content := styHead.Render(title) + "\n" + body
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Width(inner).
		Padding(0, 1).
		Render(content)
}

func netStyle(up bool) lipgloss.Style {
	if up {
		return styOnline
	}
	return styOffline
}

func signalStyle(pct int) lipgloss.Style {
	switch {
	case pct >= 60:
		return styOnline
	case pct >= 35:
		return styHead
	default:
		return styOffline
	}
}

// sparklineRange returns the min and max of vals, and whether there were any
// values to measure. A sparkline shows shape only (normalized to its own
// max); the range is what turns "something happened" into "these are the
// numbers" without needing a Y-axis.
func sparklineRange(vals []float64) (lo, hi float64, ok bool) {
	if len(vals) == 0 {
		return 0, 0, false
	}
	lo, hi = vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	return lo, hi, true
}

// sparkline maps values to block glyphs scaled to the current max.
func sparkline(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	max := 0.0
	for _, v := range vals {
		if v > max {
			max = v
		}
	}
	var b strings.Builder
	for _, v := range vals {
		idx := 0
		if max > 0 {
			idx = int(v / max * float64(len(blocks)-1))
		}
		b.WriteRune(blocks[idx])
	}
	return styOnline.Render(b.String())
}

// rateStr formats a byte/s rate as bits per second (Kbps/Mbps), as link speeds
// are conventionally quoted in bits.
func rateStr(bytesPerSec float64) string {
	bits := bytesPerSec * 8
	switch {
	case bits >= 1e6:
		return fmt.Sprintf("%.1f Mbps", bits/1e6)
	case bits >= 1e3:
		return fmt.Sprintf("%.1f Kbps", bits/1e3)
	default:
		return fmt.Sprintf("%.0f bps", bits)
	}
}

// byteStr formats a byte total in SI units (base 1000).
func byteStr(n uint64) string {
	f := float64(n)
	switch {
	case f >= 1e9:
		return fmt.Sprintf("%.2f GB", f/1e9)
	case f >= 1e6:
		return fmt.Sprintf("%.1f MB", f/1e6)
	case f >= 1e3:
		return fmt.Sprintf("%.1f KB", f/1e3)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func uptime(start time.Time) string {
	d := time.Since(start)
	h := int(d.Hours())
	mnt := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d:%02d", h, mnt, s)
}

func ipOr(ip net.IP, fallback string) string {
	if len(ip) == 0 {
		return fallback
	}
	return ip.String()
}
