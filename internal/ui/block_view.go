package ui

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/rohanthewiz/rterm/internal/model"
	"github.com/rohanthewiz/rterm/internal/vt"
)

// BlockView renders a single block (header + output).
type BlockView struct{}

// Layout renders the block.
func (bv *BlockView) Layout(gtx layout.Context, th *Theme, block *model.Block) layout.Dimensions {
	exitCode := block.ExitCode
	done := block.Done()

	// Determine left border color for failed blocks
	var borderColor color.NRGBA
	borderWidth := 0
	if done && exitCode != 0 {
		borderColor = th.ErrorColor
		borderWidth = 4
	}

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		// Left border for failed blocks
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if borderWidth == 0 {
				return layout.Dimensions{}
			}
			size := image.Point{X: gtx.Dp(unit.Dp(borderWidth)), Y: gtx.Constraints.Max.Y}
			defer clip.Rect(image.Rectangle{Max: size}).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, borderColor)
			return layout.Dimensions{Size: size}
		}),
		// Block content
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				// Header
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return bv.layoutHeader(gtx, th, block, done, exitCode)
				}),
				// Output
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return bv.layoutOutput(gtx, th, block)
				}),
				// Divider
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					size := image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(1))}
					defer clip.Rect(image.Rectangle{Max: size}).Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, th.DividerColor)
					return layout.Dimensions{Size: size}
				}),
			)
		}),
	)
}

func (bv *BlockView) layoutHeader(gtx layout.Context, th *Theme, block *model.Block, done bool, exitCode int) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, th.HeaderBG)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
						Spacing:   layout.SpaceBetween,
					}.Layout(gtx,
						// Left: CWD + command
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								// CWD
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Label(th.Material, th.FontSize, shortenPath(block.CWD))
									l.Font.Typeface = th.Mono
									l.Color = th.PromptColor
									return l.Layout(gtx)
								}),
								// Separator
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Label(th.Material, th.FontSize, " > ")
									l.Font.Typeface = th.Mono
									l.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
									return l.Layout(gtx)
								}),
								// Command
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									l := material.Label(th.Material, th.FontSize, block.Command)
									l.Font.Typeface = th.Mono
									l.Color = th.FG
									l.Font.Weight = font.Bold
									return l.Layout(gtx)
								}),
							)
						}),
						// Right: status indicator
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							var statusText string
							var statusColor color.NRGBA

							if !done {
								statusText = " ..."
								statusColor = th.RunningColor
							} else if exitCode == 0 {
								statusText = " ok"
								statusColor = th.SuccessColor
							} else {
								statusText = fmt.Sprintf(" E%d", exitCode)
								statusColor = th.ErrorColor
							}

							l := material.Label(th.Material, th.FontSize, statusText)
							l.Font.Typeface = th.Mono
							l.Color = statusColor
							l.Font.Weight = font.Bold
							return l.Layout(gtx)
						}),
					)
				},
			)
		},
	)
}

func (bv *BlockView) layoutOutput(gtx layout.Context, th *Theme, block *model.Block) layout.Dimensions {
	lines := block.OutputLines()
	if len(lines) == 0 {
		return layout.Dimensions{}
	}

	// Trim trailing empty lines
	for len(lines) > 0 && len(lines[len(lines)-1].Chars) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return layout.Dimensions{}
	}

	return layout.UniformInset(unit.Dp(6)).Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, len(lines))
			for i := range lines {
				line := lines[i]
				children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layoutStyledLine(gtx, th, line)
				})
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		},
	)
}

func layoutStyledLine(gtx layout.Context, th *Theme, line vt.StyledLine) layout.Dimensions {
	if len(line.Chars) == 0 {
		l := material.Label(th.Material, th.FontSize, " ")
		l.Font.Typeface = th.Mono
		return l.Layout(gtx)
	}

	// Group consecutive characters with the same style into spans
	type span struct {
		text  string
		style vt.Style
	}
	var spans []span
	var cur strings.Builder
	curStyle := line.Chars[0].Style
	for _, ch := range line.Chars {
		if ch.Style != curStyle {
			spans = append(spans, span{text: cur.String(), style: curStyle})
			cur.Reset()
			curStyle = ch.Style
		}
		cur.WriteRune(ch.Char)
	}
	if cur.Len() > 0 {
		spans = append(spans, span{text: cur.String(), style: curStyle})
	}

	if len(spans) == 1 {
		return layoutSpan(gtx, th, spans[0].text, spans[0].style)
	}

	children := make([]layout.FlexChild, len(spans))
	for i := range spans {
		s := spans[i]
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutSpan(gtx, th, s.text, s.style)
		})
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func layoutSpan(gtx layout.Context, th *Theme, txt string, style vt.Style) layout.Dimensions {
	l := material.Label(th.Material, th.FontSize, txt)
	l.Font.Typeface = th.Mono

	fg := th.FG
	if style.FGSet {
		fg = style.FG
	}
	if style.Dim {
		fg.A = 128
	}
	if style.Inverse {
		bg := th.BG
		if style.BGSet {
			bg = style.BG
		}
		fg = bg
	}
	l.Color = fg

	if style.Bold {
		l.Font.Weight = font.Bold
	}
	if style.Italic {
		l.Font.Style = font.Italic
	}
	return l.Layout(gtx)
}

// shortenPath abbreviates long paths.
func shortenPath(p string) string {
	if p == "" {
		return "~"
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(p, home) {
		p = "~" + p[len(home):]
	}
	parts := strings.Split(p, "/")
	if len(parts) <= 4 {
		return p
	}
	return parts[0] + "/" + parts[1] + "/.../" + strings.Join(parts[len(parts)-2:], "/")
}
