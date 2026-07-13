package namelookup

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// buildDNSQuery encodes a minimal single-question DNS message: header with
// QDCOUNT=1 and everything else zero, then the question itself.
func buildDNSQuery(name string, qtype uint16) []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint16(buf[4:6], 1) // QDCOUNT
	buf = append(buf, encodeName(name)...)
	buf = binary.BigEndian.AppendUint16(buf, qtype)
	buf = binary.BigEndian.AppendUint16(buf, 1) // QCLASS = IN
	return buf
}

func encodeName(name string) []byte {
	var buf []byte
	for _, label := range strings.Split(strings.TrimSuffix(name, "."), ".") {
		//nolint:gosec // G115: labels are our own short service names and IP octets, well under the 255 a byte holds.
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	return append(buf, 0)
}

// readName decodes a (possibly compressed) DNS name starting at offset in msg
// and returns the dotted name plus the offset immediately past it in the
// original message (not following any pointer, so a caller walking answer
// records after this name keeps a correct cursor).
func readName(msg []byte, offset int) (string, int, error) {
	var labels []string
	next := -1 // offset to resume at once we're done, set on the first pointer jump
	guard := 0
	for {
		guard++
		if guard > 128 || offset >= len(msg) {
			return "", 0, fmt.Errorf("malformed name")
		}
		length := int(msg[offset])
		if length == 0 {
			offset++
			break
		}
		if length&0xC0 == 0xC0 {
			if offset+1 >= len(msg) {
				return "", 0, fmt.Errorf("malformed pointer")
			}
			if next == -1 {
				next = offset + 2
			}
			offset = int(binary.BigEndian.Uint16(msg[offset:offset+2]) & 0x3FFF)
			continue
		}
		offset++
		if offset+length > len(msg) {
			return "", 0, fmt.Errorf("malformed label")
		}
		labels = append(labels, string(msg[offset:offset+length]))
		offset += length
	}
	if next != -1 {
		offset = next
	}
	return strings.Join(labels, "."), offset, nil
}
