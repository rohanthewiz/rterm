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
)

// EditorWidget is the command input area.
type EditorWidget struct {
	editor  widget.Editor
	history History
}

// NewEditorWidget creates a new editor.
func NewEditorWidget() *EditorWidget {
	e := &EditorWidget{}
	e.editor.SingleLine = true
	e.editor.Submit = true
	return e
}

// Update checks for history navigation and submit events, returning the
// submitted command if Enter was pressed.
func (e *EditorWidget) Update(gtx layout.Context) (string, bool) {
	// Poll for Up/Down arrow key events before the editor can consume them.
	// This gives Up/Down arrow keys terminal-style history recall semantics
	// (Up = previous command, Down = next command) regardless of cursor position.
	for {
		ev, ok := gtx.Event(
			key.Filter{Focus: &e.editor, Name: key.NameUpArrow},
			key.Filter{Focus: &e.editor, Name: key.NameDownArrow},
		)
		if !ok {
			break
		}
		ke, ok := ev.(key.Event)
		if !ok || ke.State != key.Press {
			continue
		}
		switch ke.Name {
		case key.NameUpArrow:
			if cmd, ok := e.history.Previous(e.editor.Text()); ok {
				n := len([]rune(cmd))
				e.editor.SetText(cmd)
				e.editor.SetCaret(n, n)
			}
		case key.NameDownArrow:
			if cmd, ok := e.history.Next(); ok {
				n := len([]rune(cmd))
				e.editor.SetText(cmd)
				e.editor.SetCaret(n, n)
			}
		}
	}

	// Poll editor for submitted commands.
	for {
		ev, ok := e.editor.Update(gtx)
		if !ok {
			break
		}
		if sub, ok := ev.(widget.SubmitEvent); ok {
			cmd := sub.Text
			e.editor.SetText("")
			return cmd, true
		}
	}
	return "", false
}

// AddHistory records a command in history.
func (e *EditorWidget) AddHistory(cmd string) {
	e.history.Add(cmd)
}

// Layout renders the editor.
func (e *EditorWidget) Layout(gtx layout.Context, th *Theme) layout.Dimensions {
	return layout.Background{}.Layout(gtx,
		// Background fill
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Min}).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, th.EditorBG)
			return layout.Dimensions{Size: gtx.Constraints.Min}
		},
		// Editor content
		func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(8)).Layout(gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						// Prompt character
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							l := material.Label(th.Material, th.FontSize, "> ")
							l.Font.Typeface = th.Mono
							l.Color = th.PromptColor
							return l.Layout(gtx)
						}),
						// Text input
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							ed := material.Editor(th.Material, &e.editor, "type a command...")
							ed.Font.Typeface = th.Mono
							ed.TextSize = th.FontSize
							ed.Color = th.FG
							ed.HintColor = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
							return ed.Layout(gtx)
						}),
					)
				},
			)
		},
	)
}

// SetText replaces the editor content with the given text.
func (e *EditorWidget) SetText(text string) {
	n := len([]rune(text))
	e.editor.SetText(text)
	e.editor.SetCaret(n, n)
}

// AppendText appends text to the current editor content.
func (e *EditorWidget) AppendText(text string) {
	current := e.editor.Text()
	if current != "" && !strings.HasSuffix(current, " ") {
		current += " "
	}
	combined := current + text
	n := len([]rune(combined))
	e.editor.SetText(combined)
	e.editor.SetCaret(n, n)
}

// Focus requests keyboard focus for the editor.
func (e *EditorWidget) Focus(gtx layout.Context) {
	gtx.Execute(key.FocusCmd{Tag: &e.editor})
}
