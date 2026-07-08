//go:build windows

// Package ifacestat reads an interface's cumulative byte counters, so the
// dashboard can derive live throughput from successive samples.
package ifacestat

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gsjonio/netwp/internal/core"
)

// Reader reads 64-bit byte counters for one interface via Get-NetAdapterStatistics.
//
// ponytail: shells out to PowerShell once per sample instead of binding
// iphlpapi's GetIfEntry2 (a ~1.4 KB struct of unsafe fields). Fine at the
// dashboard's 1 s cadence; switch to the native call if the per-tick process
// spawn ever shows up on the CPU.
type Reader struct {
	iface string
}

func New(iface string) Reader { return Reader{iface: iface} }

func (r Reader) Counters() (core.NetCounters, error) {
	// %ReceivedBytes and %SentBytes are ULONG64; print them space-separated.
	cmd := fmt.Sprintf(
		`$s = Get-NetAdapterStatistics -InterfaceAlias '%s'; "$($s.ReceivedBytes) $($s.SentBytes)"`,
		strings.ReplaceAll(r.iface, "'", "''"),
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", cmd).Output()
	if err != nil {
		return core.NetCounters{}, fmt.Errorf("Get-NetAdapterStatistics: %w", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return core.NetCounters{}, fmt.Errorf("unexpected stats output %q", string(out))
	}
	rx, err1 := strconv.ParseUint(fields[0], 10, 64)
	tx, err2 := strconv.ParseUint(fields[1], 10, 64)
	if err1 != nil || err2 != nil {
		return core.NetCounters{}, fmt.Errorf("parsing stats %q", string(out))
	}
	return core.NetCounters{RxBytes: rx, TxBytes: tx}, nil
}
