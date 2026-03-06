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
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/rohanthewiz/rterm/internal/model"
	"github.com/rohanthewiz/rterm/internal/vt"
)

// BlockAction represents an action triggered on a block.
type BlockAction int

const (
	BlockActionNone BlockAction = iota
	BlockActionCopy             // Copy command to clipboard
	BlockActionAppend           // Append command to input buffer
)

// blockState holds per-block UI state that persists across frames.
type blockState struct {
	collapsed    bool
	toggleClick  widget.Clickable
	copyClick    widget.Clickable
	appendClick  widget.Clickable
}

// BlockView renders a single block (header + output) with collapse/expand.
type BlockView struct{}

// BlockResult is returned from Layout with any action the user triggered.
type BlockResult struct {
	Action  BlockAction
	Command string
}

// Layout renders the block and returns any triggered action.
func (bv *BlockView) Layout(gtx layout.Context, th *Theme, block *model.Block, state *blockState, searchTerm string) (layout.Dimensions, BlockResult) {
	result := BlockResult{}

	// Check for button clicks
	if state.toggleClick.Clicked(gtx) {
		state.collapsed = !state.collapsed
	}
	if state.copyClick.Clicked(gtx) {
		result = BlockResult{Action: BlockActionCopy, Command: block.Command}
	}
	if state.appendClick.Clicked(gtx) {
		result = BlockResult{Action: BlockActionAppend, Command: block.Command}
	}

	exitCode := block.ExitCode
	done := block.Done()

	// Determine left border color for failed blocks
	var borderColor color.NRGBA
	borderWidth := 0
	if done && exitCode != 0 {
		borderColor = th.ErrorColor
		borderWidth = 4
	}

	dims := layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
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
			children := []layout.FlexChild{
				// Header (always shown)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return bv.layoutHeader(gtx, th, block, state, done, exitCode)
				}),
			}

			// Output (only when expanded and not collapsed)
			if !state.collapsed {
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return bv.layoutOutput(gtx, th, block, searchTerm)
					}),
				)
			}

			// Divider (always shown)
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					size := image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(1))}
					defer clip.Rect(image.Rectangle{Max: size}).Push(gtx.Ops).Pop()
					paint.Fill(gtx.Ops, th.DividerColor)
					return layout.Dimensions{Size: size}
				}),
			)

			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		}),
	)

	return dims, result
}

func (bv *BlockView) layoutHeader(gtx layout.Context, th *Theme, block *model.Block, state *blockState, done bool, exitCode int) layout.Dimensions {
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
						// Left: collapse toggle + CWD + command
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								// Collapse/expand toggle
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return state.toggleClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										chevron := "▼ "
										if state.collapsed {
											chevron = "▶ "
										}
										l := material.Label(th.Material, th.FontSize, chevron)
										l.Font.Typeface = th.Mono
										l.Color = th.CollapseColor
										return l.Layout(gtx)
									})
								}),
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
						// Right: action buttons + status
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
								// Copy button
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return state.copyClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											l := material.Label(th.Material, th.FontSize, " [cp]")
											l.Font.Typeface = th.Mono
											l.Color = th.ButtonColor
											return l.Layout(gtx)
										})
									})
								}),
								// Append to input button
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return state.appendClick.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
											l := material.Label(th.Material, th.FontSize, " [>>]")
											l.Font.Typeface = th.Mono
											l.Color = th.ButtonColor
											return l.Layout(gtx)
										})
									})
								}),
								// Status indicator
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
						}),
					)
				},
			)
		},
	)
}

func (bv *BlockView) layoutOutput(gtx layout.Context, th *Theme, block *model.Block, searchTerm string) layout.Dimensions {
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
					return layoutStyledLine(gtx, th, line, searchTerm)
				})
			}
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		},
	)
}

func layoutStyledLine(gtx layout.Context, th *Theme, line vt.StyledLine, searchTerm string) layout.Dimensions {
	if len(line.Chars) == 0 {
		l := material.Label(th.Material, th.FontSize, " ")
		l.Font.Typeface = th.Mono
		return l.Layout(gtx)
	}

	// Build highlight mask if searching
	var highlights []bool
	if searchTerm != "" {
		plainText := line.PlainText()
		lowerPlain := strings.ToLower(plainText)
		lowerSearch := strings.ToLower(searchTerm)
		highlights = make([]bool, len(line.Chars))

		idx := 0
		for {
			pos := strings.Index(lowerPlain[idx:], lowerSearch)
			if pos < 0 {
				break
			}
			start := idx + pos
			end := start + len(lowerSearch)
			// Map string byte positions to char positions
			charStart := byteToCharIndex(plainText, start)
			charEnd := byteToCharIndex(plainText, end)
			for j := charStart; j < charEnd && j < len(highlights); j++ {
				highlights[j] = true
			}
			idx = end
		}
	}

	// Group consecutive characters with the same style + highlight state into spans
	type span struct {
		text      string
		style     vt.Style
		highlight bool
	}
	var spans []span
	var cur strings.Builder
	curStyle := line.Chars[0].Style
	curHL := len(highlights) > 0 && highlights[0]

	for i, ch := range line.Chars {
		hl := len(highlights) > i && highlights[i]
		if ch.Style != curStyle || hl != curHL {
			spans = append(spans, span{text: cur.String(), style: curStyle, highlight: curHL})
			cur.Reset()
			curStyle = ch.Style
			curHL = hl
		}
		cur.WriteRune(ch.Char)
	}
	if cur.Len() > 0 {
		spans = append(spans, span{text: cur.String(), style: curStyle, highlight: curHL})
	}

	if len(spans) == 1 {
		return layoutSpan(gtx, th, spans[0].text, spans[0].style, spans[0].highlight)
	}

	children := make([]layout.FlexChild, len(spans))
	for i := range spans {
		s := spans[i]
		children[i] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layoutSpan(gtx, th, s.text, s.style, s.highlight)
		})
	}
	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
}

func layoutSpan(gtx layout.Context, th *Theme, txt string, style vt.Style, highlight bool) layout.Dimensions {
	if highlight {
		// Draw highlight background then text on top
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
				paint.Fill(gtx.Ops, th.SearchMatchBG)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return renderSpanText(gtx, th, txt, style)
			},
		)
	}
	return renderSpanText(gtx, th, txt, style)
}

func renderSpanText(gtx layout.Context, th *Theme, txt string, style vt.Style) layout.Dimensions {
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

// byteToCharIndex converts a byte offset in a string to a rune/char index.
func byteToCharIndex(s string, byteOffset int) int {
	charIdx := 0
	for i := range s {
		if i >= byteOffset {
			return charIdx
		}
		charIdx++
	}
	return charIdx
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
