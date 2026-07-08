//go:build linux

package ifacestat

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gsjonio/netwp/internal/core"
)

// Reader reads byte counters for one interface from /proc/net/dev.
type Reader struct {
	iface string
}

func New(iface string) Reader { return Reader{iface: iface} }

// Counters parses the line for r.iface. The /proc/net/dev format is:
//
//	iface: rxBytes rxPackets ... (8 rx fields) txBytes txPackets ... (8 tx fields)
func (r Reader) Counters() (core.NetCounters, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return core.NetCounters{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		name, rest, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(name) != r.iface {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 9 {
			return core.NetCounters{}, fmt.Errorf("short /proc/net/dev line for %s", r.iface)
		}
		rx, err1 := strconv.ParseUint(fields[0], 10, 64)
		tx, err2 := strconv.ParseUint(fields[8], 10, 64)
		if err1 != nil || err2 != nil {
			return core.NetCounters{}, fmt.Errorf("parsing /proc/net/dev for %s", r.iface)
		}
		return core.NetCounters{RxBytes: rx, TxBytes: tx}, nil
	}
	return core.NetCounters{}, fmt.Errorf("interface %s not found in /proc/net/dev", r.iface)
}
