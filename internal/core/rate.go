package core

import "time"

// NetCounters is a point-in-time reading of an interface's byte totals.
type NetCounters struct {
	RxBytes uint64
	TxBytes uint64
}

// CounterReader reads the active interface's cumulative byte counters.
type CounterReader interface {
	Counters() (NetCounters, error)
}

// Rate is a throughput reading derived from two counter samples.
type Rate struct {
	DownBps float64 // bytes per second, download
	UpBps   float64 // bytes per second, upload
	TotalRx uint64  // bytes received since the meter started
	TotalTx uint64  // bytes sent since the meter started
}

// RateMeter turns successive NetCounters samples into throughput. Not safe for
// concurrent use; drive it from one goroutine (the dashboard's update loop).
type RateMeter struct {
	started bool
	startRx uint64
	startTx uint64
	last    NetCounters
	lastAt  time.Time
}

// Update records a new sample and returns the throughput since the previous one.
// The first call establishes the baseline and reports zero rates.
func (m *RateMeter) Update(c NetCounters, now time.Time) Rate {
	if !m.started {
		m.started = true
		m.startRx, m.startTx = c.RxBytes, c.TxBytes
		m.last, m.lastAt = c, now
		return Rate{}
	}

	secs := now.Sub(m.lastAt).Seconds()
	var down, up float64
	if secs > 0 {
		// A counter that went backwards means the NIC reset or the interface
		// changed; treat that interval as no traffic rather than a huge spike.
		down = delta(c.RxBytes, m.last.RxBytes) / secs
		up = delta(c.TxBytes, m.last.TxBytes) / secs
	}
	m.last, m.lastAt = c, now

	return Rate{
		DownBps: down,
		UpBps:   up,
		TotalRx: sub(c.RxBytes, m.startRx),
		TotalTx: sub(c.TxBytes, m.startTx),
	}
}

func delta(cur, prev uint64) float64 {
	if cur < prev {
		return 0
	}
	return float64(cur - prev)
}

func sub(cur, base uint64) uint64 {
	if cur < base {
		return 0
	}
	return cur - base
}
