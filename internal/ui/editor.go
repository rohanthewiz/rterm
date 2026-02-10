package ui

import (
	"image"
	"image/color"

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
	editor widget.Editor
}

// NewEditorWidget creates a new editor.
func NewEditorWidget() *EditorWidget {
	e := &EditorWidget{}
	e.editor.SingleLine = true
	e.editor.Submit = true
	return e
}

// Update checks for submit events and returns the submitted command, if any.
func (e *EditorWidget) Update(gtx layout.Context) (string, bool) {
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

// Focus requests keyboard focus for the editor.
func (e *EditorWidget) Focus(gtx layout.Context) {
	gtx.Execute(key.FocusCmd{Tag: &e.editor})
}
