package namelookup

import (
	"encoding/binary"
	"net"
	"strings"
	"time"
)

const nbstatQType = 0x21 // NBSTAT (RFC 1002 §4.2.1.3)

// netbiosLookup asks a host directly (unicast UDP/137) for its NetBIOS
// computer name via an NBSTAT query. Windows machines and older/embedded
// devices (printers, NAS boxes) answer this even when they have no mDNS
// responder; it's the second fallback after mDNS.
func netbiosLookup(ip net.IP, timeout time.Duration) string {
	conn, err := net.DialTimeout("udp4", net.JoinHostPort(ip.String(), "137"), timeout)
	if err != nil {
		return ""
	}
	defer conn.Close() //nolint:errcheck // best-effort cleanup

	if _, err := conn.Write(nbstatQuery()); err != nil {
		return ""
	}
	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return ""
	}
	return parseNBStatName(buf[:n])
}

// nbstatQuery builds a NetBIOS Name Service NBSTAT request for the wildcard
// name "*", asking whatever answers for its own name table.
func nbstatQuery() []byte {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint16(buf[4:6], 1) // QDCOUNT = 1
	buf = append(buf, encodeNetBIOSName("*")...)
	buf = binary.BigEndian.AppendUint16(buf, nbstatQType)
	buf = binary.BigEndian.AppendUint16(buf, 1) // QCLASS = IN
	return buf
}

// encodeNetBIOSName applies NetBIOS first-level ("half-ASCII") encoding: the
// name is padded to 16 bytes (NUL-padded here, which is what the wildcard "*"
// query expects), then each byte's two nibbles become a letter 'A'+nibble.
// Framed as a single 32-byte label followed by the root label, per RFC 1002.
func encodeNetBIOSName(name string) []byte {
	raw := make([]byte, 16)
	copy(raw, name)
	encoded := make([]byte, 32)
	for i, b := range raw {
		encoded[i*2] = 'A' + (b >> 4)
		encoded[i*2+1] = 'A' + (b & 0x0F)
	}
	return append(append([]byte{32}, encoded...), 0)
}

// parseNBStatName reads the first name out of an NBSTAT response's RDATA
// name table (RFC 1002 §4.2.18) and returns it trimmed, or "" if
// absent/malformed. Reuses the general DNS name reader for the question and
// answer name fields since NBSTAT responses echo them the same way plain DNS
// does (either literally or via a compression pointer).
func parseNBStatName(msg []byte) string {
	if len(msg) < 12 {
		return ""
	}
	ancount := int(binary.BigEndian.Uint16(msg[6:8]))
	if ancount == 0 {
		return ""
	}

	_, offset, err := readName(msg, 12)
	if err != nil || offset+4 > len(msg) {
		return ""
	}
	offset += 4 // QTYPE + QCLASS

	_, offset, err = readName(msg, offset)
	if err != nil || offset+10 > len(msg) {
		return ""
	}
	offset += 10 // TYPE(2) CLASS(2) TTL(4) RDLENGTH(2)
	if offset+1 > len(msg) {
		return ""
	}

	numNames := int(msg[offset])
	offset++
	if numNames == 0 || offset+15 > len(msg) {
		return ""
	}
	name := strings.TrimSpace(string(msg[offset : offset+15]))
	return name
}
