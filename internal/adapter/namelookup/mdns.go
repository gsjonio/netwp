package namelookup

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

const mdnsAddr = "224.0.0.251:5353"
const dnsTypePTR = 12

// mdnsReverseLookup asks the local multicast-DNS group "what is your name"
// for ip via a reverse PTR query (RFC 6762 §12) and returns the first
// answer's name (".local" stripped), or "" if nothing answers within
// timeout. Works against Bonjour/Avahi-style responders: Macs, iPhones,
// Linux boxes with avahi, Chromecasts, and most smart-home gear.
//
// ponytail: one query, one short listen window, first answer wins. A real
// mDNS client caches results and answers many questions per socket; this
// fires once per unresolved device, which is enough for a fill-in-the-blanks
// pass over a scan.
func mdnsReverseLookup(ip net.IP, timeout time.Duration) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	name := fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa.", ip4[3], ip4[2], ip4[1], ip4[0])
	query := buildDNSQuery(name, dnsTypePTR)

	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return ""
	}
	defer conn.Close()

	dst, err := net.ResolveUDPAddr("udp4", mdnsAddr)
	if err != nil {
		return ""
	}
	if _, err := conn.WriteToUDP(query, dst); err != nil {
		return ""
	}

	deadline := time.Now().Add(timeout)
	buf := make([]byte, 2048)
	for {
		conn.SetReadDeadline(deadline)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			return "" // timeout: nobody answered in time
		}
		if got := parsePTRAnswer(buf[:n]); got != "" {
			return got
		}
	}
}

// parsePTRAnswer walks a DNS response's question and answer sections looking
// for the first PTR record, returning its name with a trailing ".local"
// stripped.
func parsePTRAnswer(msg []byte) string {
	if len(msg) < 12 {
		return ""
	}
	qdcount := int(binary.BigEndian.Uint16(msg[4:6]))
	ancount := int(binary.BigEndian.Uint16(msg[6:8]))
	if ancount == 0 {
		return ""
	}

	offset := 12
	for i := 0; i < qdcount; i++ {
		_, next, err := readName(msg, offset)
		if err != nil || next+4 > len(msg) {
			return ""
		}
		offset = next + 4 // QTYPE + QCLASS
	}

	for i := 0; i < ancount; i++ {
		_, next, err := readName(msg, offset)
		if err != nil || next+10 > len(msg) {
			return ""
		}
		rtype := binary.BigEndian.Uint16(msg[next : next+2])
		rdlength := int(binary.BigEndian.Uint16(msg[next+8 : next+10]))
		rdataStart := next + 10
		if rdataStart+rdlength > len(msg) {
			return ""
		}
		if rtype == dnsTypePTR {
			name, _, err := readName(msg, rdataStart)
			if err == nil && name != "" {
				return strings.TrimSuffix(name, ".local")
			}
		}
		offset = rdataStart + rdlength
	}
	return ""
}
