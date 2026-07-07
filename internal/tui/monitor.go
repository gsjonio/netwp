package tui

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"

	"github.com/gsjonio/netwp/internal/core"
)

const logLimit = 8 // recent-activity lines kept on screen

var (
	styOnline  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styOffline = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styHead    = lipgloss.NewStyle().Bold(true)
	styTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
)

// RunMonitor starts the live monitoring UI and blocks until the user quits.
// interval is the gap between scans; scanBudget bounds how long each scan may
// run (kept separate: a scan can legitimately take longer than the interval).
func RunMonitor(discovery *core.Discovery, tracker *core.Tracker, network core.Network, interval, scanBudget time.Duration) error {
	m := monitorModel{
		discovery:  discovery,
		tracker:    tracker,
		network:    network,
		interval:   interval,
		scanBudget: scanBudget,
		scanning:   true,
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

	log      []string
	lastScan time.Time
	scanning bool
	err      error
}

type scanDoneMsg struct {
	devices []core.Device
	at      time.Time
	err     error
}

type rescanMsg struct{}

func (m monitorModel) Init() tea.Cmd { return m.scanNow }

// scanNow runs one discovery pass off the UI goroutine (it is a tea.Cmd).
func (m monitorModel) scanNow() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), m.scanBudget)
	defer cancel()
	devices, err := m.discovery.Run(ctx, m.network)
	return scanDoneMsg{devices: devices, at: time.Now(), err: err}
}

func (m monitorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "r":
			if !m.scanning {
				m.scanning = true
				return m, m.scanNow
			}
		}

	case scanDoneMsg:
		m.scanning = false
		m.lastScan = msg.at
		m.err = msg.err
		if msg.err == nil {
			for _, e := range m.tracker.Observe(msg.devices, msg.at) {
				m.log = append(m.log, formatEvent(e))
			}
			if len(m.log) > logLimit {
				m.log = m.log[len(m.log)-logLimit:]
			}
		}
		return m, tea.Tick(m.interval, func(time.Time) tea.Msg { return rescanMsg{} })

	case rescanMsg:
		if !m.scanning {
			m.scanning = true
			return m, m.scanNow
		}
	}
	return m, nil
}

func (m monitorModel) View() string {
	devices := m.tracker.Devices()
	online := 0
	for _, d := range devices {
		if d.Online {
			online++
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s · %d online / %d known\n\n",
		styTitle.Render("netwp monitor"), m.network.CIDR, online, len(devices))
	b.WriteString(renderMonitorTable(devices, m.lastScan))
	b.WriteString("\n\n")

	if len(m.log) > 0 {
		b.WriteString(styHead.Render("Recent activity") + "\n")
		b.WriteString(strings.Join(m.log, "\n") + "\n\n")
	}

	state := "idle"
	if m.scanning {
		state = "scanning…"
	}
	if m.err != nil {
		b.WriteString(styOffline.Render("error: "+m.err.Error()) + "\n")
	}
	b.WriteString(styOffline.Render(fmt.Sprintf("[%s]  every %s   ·   r rescan   ·   q quit", state, m.interval)))
	return b.String()
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
		Headers("", "IP", "MAC", "HOSTNAME", "VENDOR", "LAST SEEN")

	for _, d := range devices {
		dot := styOffline.Render("○")
		if d.Online {
			dot = styOnline.Render("●")
		}
		t.Row(dot, d.IP.String(), macText(d.MAC), orDash(d.Hostname), orDash(d.Vendor), lastSeen(d, ref))
	}
	return t.String()
}

func formatEvent(e core.Event) string {
	name := e.Device.Hostname
	if name == "" {
		name = e.Device.IP.String()
	}
	ts := e.At.Format("15:04:05")
	if e.Kind == core.Joined {
		return styOnline.Render("＋") + " " + ts + "  " + name + " joined"
	}
	return styOffline.Render("－") + " " + ts + "  " + name + " left"
}

func macText(m net.HardwareAddr) string {
	if len(m) == 0 {
		return "—"
	}
	return m.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
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
