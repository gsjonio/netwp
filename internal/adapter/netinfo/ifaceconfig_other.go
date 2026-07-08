//go:build !windows

package netinfo

import (
	"errors"

	"github.com/gsjonio/netwp/internal/core"
)

// ponytail: not implemented off Windows yet. Linux would shell out to `ip
// addr`/`ip route` (or `nmcli` where NetworkManager owns the interface).
type Configurator struct{}

func (Configurator) SetStatic(cfg core.StaticConfig) error {
	return errors.New("interface configuration is not implemented on this platform yet")
}

func (Configurator) SetDHCP() error {
	return errors.New("interface configuration is not implemented on this platform yet")
}
