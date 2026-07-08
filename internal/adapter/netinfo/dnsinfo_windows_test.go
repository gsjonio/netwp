//go:build windows

package netinfo

import "testing"

func TestPSQuote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Ethernet", "Ethernet"},
		{"Wi-Fi 2", "Wi-Fi 2"},
		{"evil'; Remove-Item C:\\", "evil''; Remove-Item C:\\"},
	}
	for _, c := range cases {
		if got := psQuote(c.in); got != c.want {
			t.Errorf("psQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
