package namelookup

import (
	"encoding/binary"
	"testing"
)

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
