package core

import (
	"context"
	"time"
)

// BandwidthResult holds a speed-test outcome in megabits per second.
type BandwidthResult struct {
	DownloadMbps float64
	UploadMbps   float64
}

// SpeedTester transfers a fixed number of bytes and reports how long it took.
// Implemented by an adapter talking to a public speed-test endpoint.
type SpeedTester interface {
	Download(ctx context.Context, size int64) (time.Duration, error)
	Upload(ctx context.Context, size int64) (time.Duration, error)
}

// Speedtest is the bandwidth-measurement use case.
type Speedtest struct {
	tester        SpeedTester
	downloadBytes int64
	uploadBytes   int64
}

// NewSpeedtest builds the use case with transfer sizes tuned for a typical home
// link.
//
// ponytail: fixed sizes. Adaptive ramping (grow until the link saturates) can
// come later if the numbers read low on very fast or very slow connections.
func NewSpeedtest(tester SpeedTester) *Speedtest {
	return &Speedtest{tester: tester, downloadBytes: 25_000_000, uploadBytes: 10_000_000}
}

// Run measures download then upload throughput.
func (s *Speedtest) Run(ctx context.Context) (BandwidthResult, error) {
	down, err := s.tester.Download(ctx, s.downloadBytes)
	if err != nil {
		return BandwidthResult{}, err
	}
	up, err := s.tester.Upload(ctx, s.uploadBytes)
	if err != nil {
		return BandwidthResult{}, err
	}
	return BandwidthResult{
		DownloadMbps: Mbps(s.downloadBytes, down),
		UploadMbps:   Mbps(s.uploadBytes, up),
	}, nil
}

// Mbps converts a byte count and a duration into megabits per second.
func Mbps(bytes int64, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(bytes) * 8 / d.Seconds() / 1e6
}
