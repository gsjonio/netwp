package httpspeed

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/__down", func(w http.ResponseWriter, r *http.Request) {
		n := 1024
		if b := r.URL.Query().Get("bytes"); b != "" {
			if v, err := parseInt(b); err == nil {
				n = v
			}
		}
		time.Sleep(time.Millisecond) // ponytail: guarantees a measurable duration on fast loopback
		w.Write(make([]byte, n))
	})
	mux.HandleFunc("/__up", func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		time.Sleep(time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, io.ErrUnexpectedEOF
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

func TestDownload(t *testing.T) {
	srv := newTestServer(t)
	tester := Tester{Client: srv.Client(), BaseURL: srv.URL}

	d, err := tester.Download(context.Background(), 2048)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d <= 0 {
		t.Errorf("expected positive duration, got %v", d)
	}
}

func TestUpload(t *testing.T) {
	srv := newTestServer(t)
	tester := Tester{Client: srv.Client(), BaseURL: srv.URL}

	d, err := tester.Upload(context.Background(), 2048)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d <= 0 {
		t.Errorf("expected positive duration, got %v", d)
	}
}

// TestUploadStreamsExactBytes proves the streamed (non-allocated) payload
// still sends exactly `size` bytes, all zero.
func TestUploadStreamsExactBytes(t *testing.T) {
	var got int64
	var allZero = true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, _ := io.ReadAll(r.Body)
		got = int64(len(buf))
		for _, b := range buf {
			if b != 0 {
				allZero = false
				break
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	tester := Tester{Client: srv.Client(), BaseURL: srv.URL}

	const size = 4096
	if _, err := tester.Upload(context.Background(), size); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != size {
		t.Errorf("server received %d bytes, want %d", got, size)
	}
	if !allZero {
		t.Error("payload was not all-zero")
	}
}

func TestColo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/cdn-cgi/trace", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "fl=123abc\nh=speed.cloudflare.com\ncolo=GRU\nip=203.0.113.1\n")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	tester := Tester{Client: srv.Client(), BaseURL: srv.URL}

	if got := tester.Colo(context.Background()); got != "GRU" {
		t.Errorf("Colo() = %q, want GRU", got)
	}
}

func TestColoMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "not a trace response")
	}))
	t.Cleanup(srv.Close)
	tester := Tester{Client: srv.Client(), BaseURL: srv.URL}

	if got := tester.Colo(context.Background()); got != "" {
		t.Errorf("Colo() = %q, want empty for a response with no colo= line", got)
	}
}

func TestDownloadServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	tester := Tester{Client: srv.Client(), BaseURL: srv.URL}

	if _, err := tester.Download(context.Background(), 1024); err == nil {
		t.Error("expected error on server failure, got nil")
	}
}
