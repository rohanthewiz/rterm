package ui

import (
	"image"
	"image/color"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/rohanthewiz/rterm/internal/model"
)

// SearchWidget provides search across all blocks or within a specific block.
type SearchWidget struct {
	editor   widget.Editor
	visible  bool
	close    widget.Clickable
	matchCount int
}

// NewSearchWidget creates a new search widget.
func NewSearchWidget() *SearchWidget {
	s := &SearchWidget{}
	s.editor.SingleLine = true
	return s
}

// Toggle shows or hides the search bar.
func (s *SearchWidget) Toggle() {
	s.visible = !s.visible
	if !s.visible {
		s.editor.SetText("")
		s.matchCount = 0
	}
}

// Show makes the search bar visible.
func (s *SearchWidget) Show() {
	s.visible = true
}

// Hide hides the search bar and clears the search term.
func (s *SearchWidget) Hide() {
	s.visible = false
	s.editor.SetText("")
	s.matchCount = 0
}

// Visible reports whether the search bar is currently shown.
func (s *SearchWidget) Visible() bool {
	return s.visible
}

// Term returns the current search text.
func (s *SearchWidget) Term() string {
	if !s.visible {
		return ""
	}
	return s.editor.Text()
}

// Focus requests keyboard focus for the search editor.
func (s *SearchWidget) Focus(gtx layout.Context) {
	gtx.Execute(key.FocusCmd{Tag: &s.editor})
}

// Update processes events. Returns true if Escape was pressed (close search).
func (s *SearchWidget) Update(gtx layout.Context, session *model.Session) bool {
	if !s.visible {
		return false
	}

	if s.close.Clicked(gtx) {
		s.Hide()
		return true
	}

	// Check for Escape to close
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &s.editor, Name: key.NameEscape},
		)
		if !ok {
			break
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		if ke.Name == key.NameEscape {
			s.Hide()
			return true
		}
	}

	// Process editor events
	for {
		_, ok := s.editor.Update(gtx)
		if !ok {
			break
		}
	}

	// Count matches
	term := s.Term()
	if term != "" {
		s.matchCount = countMatches(session, term)
	} else {
		s.matchCount = 0
	}

	return false
}

// Layout renders the search bar.
func (s *SearchWidget) Layout(gtx layout.Context, th *Theme) layout.Dimensions {
	if !s.visible {
		return layout.Dimensions{}
	}

	return layout.Background{}.Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, th.SearchBG)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						// Search icon/label
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(th.Material, th.FontSize, "Search: ")
							l.Font.Typeface = th.Mono
							l.Color = th.PromptColor
							return l.Layout(gtx)
						}),
						// Search input
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							ed := material.Editor(th.Material, &s.editor, "type to search...")
							ed.Font.Typeface = th.Mono
							ed.TextSize = th.FontSize
							ed.Color = th.FG
							ed.HintColor = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
							return ed.Layout(gtx)
						}),
						// Match count
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							term := s.Term()
							if term == "" {
								return layout.Dimensions{}
							}
							text := "no matches"
							if s.matchCount > 0 {
								text = strings.Join([]string{intToStr(s.matchCount), " match"}, "")
								if s.matchCount != 1 {
									text += "es"
								}
							}
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								l := material.Label(th.Material, th.FontSize, text)
								l.Font.Typeface = th.Mono
								l.Color = th.ButtonColor
								return l.Layout(gtx)
							})
						}),
						// Close button
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return s.close.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									l := material.Label(th.Material, th.FontSize, " [x]")
									l.Font.Typeface = th.Mono
									l.Color = th.ButtonColor
									return l.Layout(gtx)
								})
							})
						}),
					)
				},
			)
		},
	)
}

// countMatches counts the total number of search matches across all blocks.
func countMatches(session *model.Session, term string) int {
	if term == "" {
		return 0
	}
	lowerTerm := strings.ToLower(term)
	count := 0
	for _, block := range session.Blocks() {
		// Search in command
		count += strings.Count(strings.ToLower(block.Command), lowerTerm)
		// Search in output
		count += strings.Count(strings.ToLower(block.PlainOutput()), lowerTerm)
	}
	return count
}

// intToStr converts an int to string without importing strconv.
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
