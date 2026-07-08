package tui

import (
	"fmt"
	"io"

	"github.com/gsjonio/netwp/internal/core"
)

// RenderBandwidth prints a speed-test result in the same style as the device table.
func RenderBandwidth(w io.Writer, r core.BandwidthResult) {
	fmt.Fprintf(w, "%sdownload:%s %.1f Mbps\n", colorBold, colorReset, r.DownloadMbps)
	fmt.Fprintf(w, "%supload:  %s %.1f Mbps\n", colorBold, colorReset, r.UploadMbps)
}
