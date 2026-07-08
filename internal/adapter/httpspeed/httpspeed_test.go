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
