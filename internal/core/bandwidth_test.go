package core

import (
	"context"
	"testing"
	"time"
)

func TestMbps(t *testing.T) {
	cases := []struct {
		name  string
		bytes int64
		d     time.Duration
		want  float64
	}{
		{"1MB in 1s = 8Mbps", 1_000_000, time.Second, 8},
		{"25MB in 2s = 100Mbps", 25_000_000, 2 * time.Second, 100},
		{"zero duration", 1_000_000, 0, 0},
		{"negative duration", 1_000_000, -time.Second, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Mbps(c.bytes, c.d); got != c.want {
				t.Errorf("Mbps(%d, %v) = %v, want %v", c.bytes, c.d, got, c.want)
			}
		})
	}
}

type fakeTester struct {
	downloadFor time.Duration
	uploadFor   time.Duration
}

func (f fakeTester) Download(ctx context.Context, size int64) (time.Duration, error) {
	return f.downloadFor, nil
}

func (f fakeTester) Upload(ctx context.Context, size int64) (time.Duration, error) {
	return f.uploadFor, nil
}

func TestSpeedtestRun(t *testing.T) {
	tester := fakeTester{downloadFor: 2 * time.Second, uploadFor: 4 * time.Second}
	st := NewSpeedtest(tester)

	result, err := st.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantDown := Mbps(st.downloadBytes, tester.downloadFor)
	wantUp := Mbps(st.uploadBytes, tester.uploadFor)
	if result.DownloadMbps != wantDown {
		t.Errorf("DownloadMbps = %v, want %v", result.DownloadMbps, wantDown)
	}
	if result.UploadMbps != wantUp {
		t.Errorf("UploadMbps = %v, want %v", result.UploadMbps, wantUp)
	}
}
