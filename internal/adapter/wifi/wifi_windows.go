//go:build windows

package wifi

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/gsjonio/netwp/internal/core"
)

// Inspector implements core.WiFiInspector via netsh wlan.
//
// UNVERIFIED (connected path): the disconnected state and the visible-networks
// scan are tested against a real machine, but the connected-link fields
// (SSID/signal/channel of our own association) are only covered by fixtures,
// since the dev machine's Wi-Fi was off. Verify once connected to Wi-Fi.
type Inspector struct{}

func New() Inspector { return Inspector{} }

// netsh runs `netsh wlan <args>` through PowerShell with UTF-8 output, so the
// localized (accented) field labels come back decodable instead of in the
// console's OEM codepage.
func netsh(args string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command",
		"[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; netsh wlan "+args)
	out, err := cmd.Output()
	return string(out), err
}

func (Inspector) WiFi() (core.WiFiInfo, error) {
	out, err := netsh("show interfaces")
	if err != nil {
		return core.WiFiInfo{}, err
	}
	// No wireless hardware at all: report that distinctly.
	low := strings.ToLower(out)
	if strings.Contains(low, "no wireless interface") || strings.Contains(low, "nenhuma interface") {
		return core.WiFiInfo{}, errors.New("no wireless interface")
	}

	info := parseInterfaces(out)
	// The visible-networks scan works even while disconnected, so always attach
	// it for interference context.
	if nets, err := netsh("show networks mode=bssid"); err == nil {
		info.Nearby = parseNetworks(nets)
	}
	return info, nil
}
