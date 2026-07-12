package namelookup

import "testing"

// These fuzz targets exercise the DNS/mDNS wire parsers against arbitrary bytes:
// the responses come off the network from untrusted hosts, so a malformed packet
// must never panic or hang the scan. Run the full fuzzer with:
//
//	go test ./internal/adapter/namelookup -run=x -fuzz=FuzzReadName
//
// Under a plain `go test` only the seed corpus runs, which is a fast smoke test.

func FuzzReadName(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{3, 'a', 'b', 'c', 0})
	f.Add([]byte{0xC0, 0x00}) // compression pointer to offset 0 (self-reference)
	f.Fuzz(func(t *testing.T, msg []byte) {
		name, next, err := readName(msg, 0)
		// Contract: on success the resume offset stays within the message, so a
		// caller walking records after the name can't slice out of bounds.
		if err == nil && (next < 0 || next > len(msg)) {
			t.Fatalf("readName next=%d out of range for len=%d (name=%q)", next, len(msg), name)
		}
	})
}

func FuzzParsePTRAnswer(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 12)) // header only, zero counts
	f.Fuzz(func(t *testing.T, msg []byte) {
		_ = parsePTRAnswer(msg) // must not panic or hang
	})
}

func FuzzPtrServiceLabels(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, 12))
	f.Fuzz(func(t *testing.T, msg []byte) {
		_ = ptrServiceLabels(msg)
	})
}
