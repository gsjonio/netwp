// Package httpspeed implements core.SpeedTester against Cloudflare's public
// speed-test endpoint. No API key or self-hosted server required.
package httpspeed

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://speed.cloudflare.com"

// Tester times HTTP transfers against a speed-test server.
type Tester struct {
	Client  *http.Client
	BaseURL string // overridable for tests; defaults to Cloudflare in New().
}

func New() Tester {
	return Tester{Client: &http.Client{}, BaseURL: defaultBaseURL}
}

// Download fetches size bytes and reports how long the transfer took.
func (t Tester) Download(ctx context.Context, size int64) (time.Duration, error) {
	url := fmt.Sprintf("%s/__down?bytes=%d", t.BaseURL, size)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	start := time.Now()
	resp, err := t.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("download failed: status %d", resp.StatusCode)
	}
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

// Upload sends size bytes and reports how long the transfer took.
func (t Tester) Upload(ctx context.Context, size int64) (time.Duration, error) {
	// ponytail: an all-zero payload is fine, the endpoint only measures
	// transfer time, not entropy.
	payload := bytes.NewReader(make([]byte, size))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.BaseURL+"/__up", payload)
	if err != nil {
		return 0, err
	}
	req.ContentLength = size

	start := time.Now()
	resp, err := t.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("upload failed: status %d", resp.StatusCode)
	}
	return time.Since(start), nil
}
