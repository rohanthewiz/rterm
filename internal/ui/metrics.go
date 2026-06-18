package ui

import (
	"image"
	"strings"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
)

// CellSize measures the pixel size of a single monospace character cell for the
// theme's font and size. The layout it performs is recorded and discarded, so
// it produces no visible output.
func (th *Theme) CellSize(gtx layout.Context) image.Point {
	const n = 80 // average over many glyphs for sub-pixel accuracy

	gtx.Constraints.Min = image.Point{}
	gtx.Constraints.Max = image.Point{X: 1 << 20, Y: 1 << 20}

	l := material.Label(th.Material, th.FontSize, strings.Repeat("M", n))
	l.Font.Typeface = th.Mono

	rec := op.Record(gtx.Ops)
	dims := l.Layout(gtx)
	rec.Stop() // discard the recorded ops; this is a measurement only

	w := max(dims.Size.X/n, 1)
	h := max(dims.Size.Y, 1)
	return image.Point{X: w, Y: h}
}
