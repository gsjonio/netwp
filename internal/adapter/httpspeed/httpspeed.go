// Package httpspeed implements core.SpeedTester against Cloudflare's public
// speed-test endpoint. No API key or self-hosted server required.
package httpspeed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// zeroReader yields an endless stream of zero bytes. Wrapped in an
// io.LimitReader it feeds the upload payload without allocating the whole
// thing (uploadBytes is 10 MB): the endpoint only times the transfer, so the
// content doesn't matter.
type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

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

// Colo returns the three-letter code of the Cloudflare datacenter that
// answered (e.g. "GRU" for São Paulo), or "" if it can't be determined.
// speed.cloudflare.com is anycast: the same URL always routes to whichever of
// Cloudflare's ~300 edges is closest, so this exists to make that concrete
// instead of asserting it — hits Cloudflare's public trace endpoint, which
// every Cloudflare-proxied hostname serves.
func (t Tester) Colo(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.BaseURL+"/cdn-cgi/trace", nil)
	if err != nil {
		return ""
	}
	resp, err := t.Client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(body), "\n") {
		if colo, ok := strings.CutPrefix(line, "colo="); ok {
			return strings.TrimSpace(colo)
		}
	}
	return ""
}

// Upload sends size bytes and reports how long the transfer took.
func (t Tester) Upload(ctx context.Context, size int64) (time.Duration, error) {
	// ponytail: an all-zero payload is fine, the endpoint only measures
	// transfer time, not entropy. Stream it (see zeroReader) instead of
	// allocating `size` bytes up front.
	payload := io.LimitReader(zeroReader{}, size)
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
