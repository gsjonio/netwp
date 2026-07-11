package namelookup

import (
	"context"
	"encoding/binary"
	"net"
	"strings"
	"time"
)

// mdnsServiceWindow is how long the DNS-SD sweep listens for responses after
// firing its queries. One short window for the whole network, not per device.
const mdnsServiceWindow = 1200 * time.Millisecond

// discriminantServices are the DNS-SD service types worth asking about: each
// maps cleanly to a device class (see core.serviceClass). Kept short on
// purpose -- one multicast query per entry.
var discriminantServices = []string{
	"_googlecast._tcp.local.",      // Chromecast, Google/Nest speakers
	"_airplay._tcp.local.",         // Apple TV, AirPlay speakers
	"_raop._tcp.local.",            // AirPlay audio
	"_spotify-connect._tcp.local.", // Spotify-capable speakers
	"_amzn-wplay._tcp.local.",      // Amazon Echo/Fire
	"_ipp._tcp.local.",             // network printers
	"_ipps._tcp.local.",
	"_printer._tcp.local.",
	"_pdl-datastream._tcp.local.",
	"_companion-link._tcp.local.", // iPhone/iPad/Mac
	"_apple-mobdev2._tcp.local.",  // iPhone/iPad
	"_hap._tcp.local.",            // HomeKit accessories (smart home)
}

// ServiceScanner implements core.ServiceScanner via a single mDNS DNS-SD sweep.
type ServiceScanner struct{}

func NewServiceScanner() ServiceScanner { return ServiceScanner{} }

// Services broadcasts a PTR query for each discriminant service type, listens
// briefly, and maps each responding host's IP to the service labels it
// answered for (e.g. "192.168.1.5" -> ["_googlecast"]).
//
// ponytail: the responder's source address is the device, so this reads the
// service type off the PTR answer's own name and skips the SRV/A chasing a
// full DNS-SD resolver does. Best-effort classification input: a device with
// no mDNS responder simply contributes nothing.
func (ServiceScanner) Services(ctx context.Context) map[string][]string {
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		return nil
	}
	defer conn.Close() //nolint:errcheck // best-effort cleanup

	dst, err := net.ResolveUDPAddr("udp4", mdnsAddr)
	if err != nil {
		return nil
	}
	for _, svc := range discriminantServices {
		if _, err := conn.WriteToUDP(buildDNSQuery(svc, dnsTypePTR), dst); err != nil {
			return nil
		}
	}

	deadline := time.Now().Add(mdnsServiceWindow)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	found := map[string]map[string]bool{}
	buf := make([]byte, 2048)
	for {
		_ = conn.SetReadDeadline(deadline) //nolint:errcheck // ReadFromUDP fails the same way
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // window elapsed
		}
		for _, svc := range ptrServiceLabels(buf[:n]) {
			ip := src.IP.String()
			if found[ip] == nil {
				found[ip] = map[string]bool{}
			}
			found[ip][svc] = true
		}
	}

	out := make(map[string][]string, len(found))
	for ip, set := range found {
		for svc := range set {
			out[ip] = append(out[ip], svc)
		}
	}
	return out
}

// ptrServiceLabels walks a DNS response's answer section and returns the first
// label (e.g. "_googlecast") of each PTR record's own name -- the service type
// the record advertises. Malformed input yields whatever was parsed so far.
func ptrServiceLabels(msg []byte) []string {
	if len(msg) < 12 {
		return nil
	}
	qdcount := int(binary.BigEndian.Uint16(msg[4:6]))
	ancount := int(binary.BigEndian.Uint16(msg[6:8]))

	offset := 12
	for i := 0; i < qdcount; i++ {
		_, next, err := readName(msg, offset)
		if err != nil || next+4 > len(msg) {
			return nil
		}
		offset = next + 4 // QTYPE + QCLASS
	}

	var out []string
	for i := 0; i < ancount; i++ {
		name, next, err := readName(msg, offset)
		if err != nil || next+10 > len(msg) {
			return out
		}
		rtype := binary.BigEndian.Uint16(msg[next : next+2])
		rdlength := int(binary.BigEndian.Uint16(msg[next+8 : next+10]))
		if rtype == dnsTypePTR {
			if label := firstServiceLabel(name); label != "" {
				out = append(out, label)
			}
		}
		offset = next + 10 + rdlength
	}
	return out
}

// firstServiceLabel returns the leading label of a service name if it looks
// like a DNS-SD service type ("_googlecast._tcp.local" -> "_googlecast"), else "".
func firstServiceLabel(name string) string {
	label := name
	if i := strings.Index(name, "."); i >= 0 {
		label = name[:i]
	}
	if strings.HasPrefix(label, "_") && label != "_tcp" && label != "_udp" {
		return strings.ToLower(label)
	}
	return ""
}
