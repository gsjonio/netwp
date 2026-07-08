//go:build !windows

package wifi

import (
	"errors"

	"github.com/gsjonio/netwp/internal/core"
)

// Inspector is a stub off Windows.
//
// ponytail: Linux Wi-Fi not implemented yet. Real version would parse
// `nmcli -t -f ...` (or `iw dev <if> link` + `iw dev <if> scan`) and reuse the
// same core.WiFiInfo shape. The dashboard treats the error as "no Wi-Fi panel".
type Inspector struct{}

func New() Inspector { return Inspector{} }

func (Inspector) WiFi() (core.WiFiInfo, error) {
	return core.WiFiInfo{}, errors.New("wifi info not implemented on this platform yet")
}
