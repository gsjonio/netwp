package namelookup

import (
	"encoding/binary"
	"testing"
)

func namePointer(offset int) []byte {
	return []byte{0xC0 | byte(offset>>8), byte(offset)}
}

func TestEncodeReadNameRoundTrip(t *testing.T) {
	encoded := encodeName("foo.bar.local.")
	got, next, err := readName(encoded, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "foo.bar.local" {
		t.Errorf("got %q, want foo.bar.local", got)
	}
	if next != len(encoded) {
		t.Errorf("next = %d, want %d", next, len(encoded))
	}
}

func TestReadNameWithPointer(t *testing.T) {
	msg := encodeName("example.local.")
	pointerOffset := len(msg)
	msg = append(msg, namePointer(0)...)

	got, next, err := readName(msg, pointerOffset)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "example.local" {
		t.Errorf("got %q, want example.local", got)
	}
	if next != pointerOffset+2 {
		t.Errorf("next = %d, want %d (pointer is 2 bytes, doesn't follow the jump)", next, pointerOffset+2)
	}
}

// TestParsePTRAnswer builds a synthetic mDNS reverse-lookup response by hand
// (question echoed, answer name compressed via a pointer to it) and checks
// parsePTRAnswer extracts the hostname with ".local" stripped.
func TestParsePTRAnswer(t *testing.T) {
	msg := make([]byte, 12)
	binary.BigEndian.PutUint16(msg[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(msg[6:8], 1) // ANCOUNT

	questionOffset := len(msg)
	msg = append(msg, encodeName("5.1.168.192.in-addr.arpa.")...)
	msg = binary.BigEndian.AppendUint16(msg, dnsTypePTR)
	msg = binary.BigEndian.AppendUint16(msg, 1) // QCLASS

	msg = append(msg, namePointer(questionOffset)...)
	msg = binary.BigEndian.AppendUint16(msg, dnsTypePTR)
	msg = binary.BigEndian.AppendUint16(msg, 1)   // CLASS
	msg = binary.BigEndian.AppendUint32(msg, 120) // TTL
	rdata := encodeName("kitchen-tv.local.")
	msg = binary.BigEndian.AppendUint16(msg, uint16(len(rdata)))
	msg = append(msg, rdata...)

	if got := parsePTRAnswer(msg); got != "kitchen-tv" {
		t.Errorf("got %q, want kitchen-tv", got)
	}
}

func TestParsePTRAnswerNoAnswers(t *testing.T) {
	msg := make([]byte, 12)
	binary.BigEndian.PutUint16(msg[4:6], 1) // QDCOUNT, ANCOUNT stays 0
	if got := parsePTRAnswer(msg); got != "" {
		t.Errorf("got %q, want empty for a response with no answers", got)
	}
}

// TestParseNBStatName builds a synthetic NBSTAT response (question echoed via
// a compression pointer, one name in the RDATA name table) and checks
// parseNBStatName extracts and trims it.
func TestParseNBStatName(t *testing.T) {
	msg := make([]byte, 12)
	binary.BigEndian.PutUint16(msg[4:6], 1) // QDCOUNT
	binary.BigEndian.PutUint16(msg[6:8], 1) // ANCOUNT

	questionOffset := len(msg)
	msg = append(msg, encodeNetBIOSName("*")...)
	msg = binary.BigEndian.AppendUint16(msg, nbstatQType)
	msg = binary.BigEndian.AppendUint16(msg, 1)

	msg = append(msg, namePointer(questionOffset)...)
	msg = binary.BigEndian.AppendUint16(msg, nbstatQType)
	msg = binary.BigEndian.AppendUint16(msg, 1)
	msg = binary.BigEndian.AppendUint32(msg, 0) // TTL

	rdata := []byte{1}                                  // NUM_NAMES
	rdata = append(rdata, []byte("MYPC           ")...) // 15 bytes, space-padded
	rdata = append(rdata, 0x00)                         // suffix (workstation)
	rdata = append(rdata, 0x04, 0x00)                   // flags
	msg = binary.BigEndian.AppendUint16(msg, uint16(len(rdata)))
	msg = append(msg, rdata...)

	if got := parseNBStatName(msg); got != "MYPC" {
		t.Errorf("got %q, want MYPC", got)
	}
}
