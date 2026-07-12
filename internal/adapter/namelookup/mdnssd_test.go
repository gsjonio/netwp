package namelookup

import (
	"context"
	"encoding/binary"
	"testing"
	"time"
)

// TestServiceScannerCachesSweep proves a second Services call within the TTL is
// served from cache instead of running the multicast sweep again.
func TestServiceScannerCachesSweep(t *testing.T) {
	sweeps := 0
	s := ServiceScanner{
		cache: newServiceCache(time.Minute),
		sweep: func(context.Context) map[string][]string {
			sweeps++
			return map[string][]string{"192.168.1.5": {"_googlecast"}}
		},
	}

	if got := s.Services(context.Background()); len(got["192.168.1.5"]) != 1 {
		t.Fatalf("first sweep result = %v", got)
	}
	if got := s.Services(context.Background()); len(got["192.168.1.5"]) != 1 {
		t.Fatalf("second (cached) result = %v", got)
	}
	if sweeps != 1 {
		t.Errorf("swept %d times, want 1 (second call should hit the cache)", sweeps)
	}
}

// TestServiceScannerRetriesAfterFailure checks a failed sweep (nil) is not
// cached, so the next call sweeps again.
func TestServiceScannerRetriesAfterFailure(t *testing.T) {
	sweeps := 0
	s := ServiceScanner{
		cache: newServiceCache(time.Minute),
		sweep: func(context.Context) map[string][]string {
			sweeps++
			return nil // simulate a socket error
		},
	}
	s.Services(context.Background())
	s.Services(context.Background())
	if sweeps != 2 {
		t.Errorf("swept %d times, want 2 (a nil result must not be cached)", sweeps)
	}
}

func TestServiceCacheExpiry(t *testing.T) {
	c := newServiceCache(time.Minute)
	if _, ok := c.get(); ok {
		t.Error("empty cache should miss")
	}
	c.put(map[string][]string{"a": {"_ipp"}})
	if _, ok := c.get(); !ok {
		t.Error("fresh entry should hit")
	}
	c.expires = time.Now().Add(-time.Second) // force expiry
	if _, ok := c.get(); ok {
		t.Error("expired entry should miss")
	}
}

// TestPtrServiceLabels builds a minimal mDNS response carrying one PTR answer
// for _googlecast._tcp.local and checks the leading service label is extracted.
func TestPtrServiceLabels(t *testing.T) {
	msg := make([]byte, 12)
	binary.BigEndian.PutUint16(msg[6:8], 1) // ANCOUNT = 1

	msg = append(msg, encodeName("_googlecast._tcp.local.")...)
	msg = binary.BigEndian.AppendUint16(msg, dnsTypePTR)
	msg = binary.BigEndian.AppendUint16(msg, 1)   // CLASS = IN
	msg = binary.BigEndian.AppendUint32(msg, 120) // TTL
	rdata := encodeName("Chromecast-abc._googlecast._tcp.local.")
	msg = binary.BigEndian.AppendUint16(msg, uint16(len(rdata)))
	msg = append(msg, rdata...)

	got := ptrServiceLabels(msg)
	if len(got) != 1 || got[0] != "_googlecast" {
		t.Errorf("ptrServiceLabels = %v, want [_googlecast]", got)
	}
}

func TestFirstServiceLabel(t *testing.T) {
	cases := map[string]string{
		"_ipp._tcp.local": "_ipp",
		"_HAP._tcp.local": "_hap", // lowercased
		"host.local":      "",     // not a service type
		"_tcp":            "",     // structural label, not a service
	}
	for in, want := range cases {
		if got := firstServiceLabel(in); got != want {
			t.Errorf("firstServiceLabel(%q) = %q, want %q", in, got, want)
		}
	}
}
