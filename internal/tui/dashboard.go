package tui

import (
	"context"
	"fmt"
	"net"
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
	histLen         = 24
)

// RunDashboard starts the composite live dashboard and blocks until the user quits.
func RunDashboard(discovery *core.Discovery, tracker *core.Tracker, network core.Network,
	info core.InterfaceInfo, reader core.CounterReader, wifi core.WiFiInspector, speed *core.Speedtest, pinger core.Pinger) error {
	m := dashModel{
		discovery: discovery,
		tracker:   tracker,
		network:   network,
		info:      info,
		reader:    reader,
		wifi:      wifi,
		speed:     speed,
		pinger:    pinger,
		meter:     &core.RateMeter{},
		start:     time.Now(),
		width:     dashDefaultCols,
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
	meter     *core.RateMeter

	start      time.Time
	rate       core.Rate
	downHist   []float64
	wifiInfo   core.WiFiInfo
	wifiErr    error
	result     core.BandwidthResult
	speedAt    time.Time
	speedErr   error
	netLatency time.Duration
	netUp      bool
	lastScan   time.Time
	log        []string
	width      int
	height     int
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
	devices, _ := m.discovery.Run(ctx, m.network)
	return scanMsg{devices: devices, at: time.Now()}
}

func (m dashModel) runSpeed() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), dashSpeedBudget)
	defer cancel()
	r, err := m.speed.Run(ctx)
	return speedMsg{result: r, at: time.Now(), err: err}
}

func (m dashModel) readPing() tea.Msg {
	rtt, ok := m.pinger.Ping(net.ParseIP(dashPingTarget), 800*time.Millisecond)
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
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			return m, m.scan
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case sampleMsg:
		m.rate = m.meter.Update(msg.c, time.Now())
		m.downHist = append(m.downHist, m.rate.DownBps)
		if len(m.downHist) > histLen {
			m.downHist = m.downHist[len(m.downHist)-histLen:]
		}
		return m, tick(dashSampleEvery, sampleTickMsg{})
	case sampleTickMsg:
		return m, m.readSample

	case wifiMsg:
		m.wifiInfo, m.wifiErr = msg.info, msg.err
		return m, tick(dashWifiEvery, wifiTickMsg{})
	case wifiTickMsg:
		return m, m.readWifi

	case scanMsg:
		m.lastScan = msg.at
		for _, e := range m.tracker.Observe(msg.devices, msg.at) {
			m.log = append(m.log, formatEvent(e))
		}
		if len(m.log) > logLimit {
			m.log = m.log[len(m.log)-logLimit:]
		}
		return m, tick(dashScanEvery, scanTickMsg{})
	case scanTickMsg:
		return m, m.scan

	case speedMsg:
		m.result, m.speedAt, m.speedErr = msg.result, msg.at, msg.err
		return m, tick(dashSpeedEvery, speedTickMsg{})
	case speedTickMsg:
		return m, m.runSpeed

	case pingMsg:
		m.netLatency, m.netUp = msg.rtt, msg.ok
		return m, tick(dashPingEvery, pingTickMsg{})
	case pingTickMsg:
		return m, m.readPing
	}
	return m, nil
}

func (m dashModel) View() string {
	width := m.width
	if width < 60 {
		width = dashDefaultCols
	}

	header := m.renderHeader(width)
	colW := (width - 4) / 3
	top := lipgloss.JoinHorizontal(lipgloss.Top,
		panel("WI-FI", m.renderWifi(), colW),
		panel("BANDWIDTH", m.renderBandwidth(), colW),
		panel("SPEEDTEST", m.renderSpeed(), colW),
	)
	footer := styOffline.Render("r rescan · q quit")

	var activity string
	if len(m.log) > 0 {
		activity = panel("ACTIVITY", strings.Join(m.log, "\n"), width-2)
	}

	devices := m.tracker.Devices()
	total := len(devices)
	online := 0
	for _, d := range devices {
		if d.Online {
			online++
		}
	}
	devTitle := fmt.Sprintf("DEVICES · %d online / %d known", online, total)

	// Trim the device table to whatever vertical room is left, so the footer
	// never scrolls off screen on a short terminal. Fixed overhead below the
	// header/top/activity/footer lines: the device panel's own border (2) +
	// title (1), plus the inner table's border+header+separator+border (4).
	if m.height > 0 {
		used := lineCount(header) + lineCount(top) + lineCount(footer) + 7
		if activity != "" {
			used += lineCount(activity)
		}
		if budget := m.height - used; budget > 0 && total > budget {
			devices = devices[:budget]
			devTitle = fmt.Sprintf("DEVICES · %d online / %d known (showing %d)", online, total, budget)
		}
	}
	devBody := renderMonitorTable(devices, m.lastScan)

	parts := []string{header, top}
	if activity != "" {
		parts = append(parts, activity)
	}
	parts = append(parts, panel(devTitle, devBody, width-2), footer)
	return strings.Join(parts, "\n")
}

func lineCount(s string) int { return strings.Count(s, "\n") + 1 }

func (m dashModel) renderHeader(width int) string {
	title := styTitle.Render("netwp dashboard")
	clock := time.Now().Format("15:04:05")
	left := fmt.Sprintf("%s   %s · IP %s · GW %s · net %s · up %s",
		title, m.info.Name, m.info.IP, ipOr(m.info.Gateway, "?"),
		netStyle(m.netUp).Render(rttText(m.netLatency, m.netUp)), uptime(m.start))
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
	lines := []string{
		styAlias.Render(w.SSID),
		"signal   " + signalStyle(w.SignalPercent).Render(sig),
		fmt.Sprintf("channel  %d  %s  %s", w.Channel, w.Band, channelHint),
		fmt.Sprintf("rate     %d/%d Mbps", w.RxRateMbps, w.TxRateMbps),
		fmt.Sprintf("nearby   %d APs · %d on ch %d", len(w.Nearby), w.SameChannelCount(), w.Channel),
	}
	return strings.Join(lines, "\n")
}

func (m dashModel) renderBandwidth() string {
	lines := []string{
		"down  " + styOnline.Render(rateStr(m.rate.DownBps)),
		"up    " + styHead.Render(rateStr(m.rate.UpBps)),
		sparkline(m.downHist),
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
	return strings.Join([]string{
		fmt.Sprintf("down  %s", styOnline.Render(fmt.Sprintf("%.1f Mbps", m.result.DownloadMbps))),
		fmt.Sprintf("up    %s", styHead.Render(fmt.Sprintf("%.1f Mbps", m.result.UploadMbps))),
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
