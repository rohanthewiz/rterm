package ui

import (
	"image/color"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

// Theme holds terminal color and font settings.
type Theme struct {
	Material *material.Theme

	// Terminal colors
	BG           color.NRGBA
	FG           color.NRGBA
	HeaderBG     color.NRGBA
	EditorBG     color.NRGBA
	DividerColor color.NRGBA
	SuccessColor color.NRGBA
	ErrorColor   color.NRGBA
	RunningColor color.NRGBA
	PromptColor  color.NRGBA

	// Block action buttons
	ButtonColor    color.NRGBA
	ButtonHover    color.NRGBA
	SearchBG       color.NRGBA
	SearchMatchBG  color.NRGBA
	CollapseColor  color.NRGBA

	// Text sizing
	FontSize unit.Sp
	Mono     font.Typeface
}

// NewTheme creates the default dark terminal theme.
func NewTheme() *Theme {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	return &Theme{
		Material: th,

		BG:           color.NRGBA{R: 30, G: 30, B: 30, A: 255},
		FG:           color.NRGBA{R: 204, G: 204, B: 204, A: 255},
		HeaderBG:     color.NRGBA{R: 42, G: 42, B: 42, A: 255},
		EditorBG:     color.NRGBA{R: 38, G: 38, B: 38, A: 255},
		DividerColor: color.NRGBA{R: 60, G: 60, B: 60, A: 255},
		SuccessColor: color.NRGBA{R: 85, G: 255, B: 85, A: 255},
		ErrorColor:   color.NRGBA{R: 255, G: 85, B: 85, A: 255},
		RunningColor: color.NRGBA{R: 255, G: 255, B: 85, A: 255},
		PromptColor:  color.NRGBA{R: 85, G: 255, B: 255, A: 255},

		ButtonColor:   color.NRGBA{R: 130, G: 130, B: 130, A: 255},
		ButtonHover:   color.NRGBA{R: 200, G: 200, B: 200, A: 255},
		SearchBG:      color.NRGBA{R: 50, G: 50, B: 50, A: 255},
		SearchMatchBG: color.NRGBA{R: 100, G: 80, B: 0, A: 255},
		CollapseColor: color.NRGBA{R: 130, G: 130, B: 130, A: 255},

		FontSize: unit.Sp(14),
		Mono:     "Go Mono",
	}
}
