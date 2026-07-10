package core

// DiffResult buckets what changed between two scan snapshots, keyed by MAC
// (a device's stable identity across a DHCP-assigned IP change).
type DiffResult struct {
	Joined     []Device // MAC not present in the previous snapshot
	Left       []Device // MAC from the previous snapshot missing now
	IPChanged  []Device // same MAC as before, now at a different IP (ordinary DHCP re-lease)
	MACChanged []Device // same IP as before, now answered by a different MAC (address takeover)
	DupMAC     []Device // MAC shared by more than one IP within the current scan
}

// Diff compares a previous scan snapshot against the current one. Devices
// with no MAC (never fully resolved) are ignored: without a MAC there is no
// stable identity to diff against.
func Diff(previous, current []Device) DiffResult {
	prevByMAC := make(map[string]Device, len(previous))
	prevByIP := make(map[string]Device, len(previous))
	for _, d := range previous {
		if len(d.MAC) == 0 {
			continue
		}
		prevByMAC[d.MAC.String()] = d
		prevByIP[d.IP.String()] = d
	}

	macCount := make(map[string]int, len(current))
	for _, d := range current {
		if len(d.MAC) > 0 {
			macCount[d.MAC.String()]++
		}
	}

	var r DiffResult
	currMACs := make(map[string]bool, len(current))
	for _, d := range current {
		if len(d.MAC) == 0 {
			continue
		}
		mac := d.MAC.String()
		currMACs[mac] = true

		if prev, ok := prevByMAC[mac]; !ok {
			r.Joined = append(r.Joined, d)
		} else if !prev.IP.Equal(d.IP) {
			r.IPChanged = append(r.IPChanged, d)
		}

		if prev, ok := prevByIP[d.IP.String()]; ok && prev.MAC.String() != mac {
			r.MACChanged = append(r.MACChanged, d)
		}
		if macCount[mac] > 1 {
			r.DupMAC = append(r.DupMAC, d)
		}
	}
	for _, d := range previous {
		if len(d.MAC) > 0 && !currMACs[d.MAC.String()] {
			r.Left = append(r.Left, d)
		}
	}
	return r
}
