package vt

import "image/color"

// Style represents text styling attributes from ANSI escape sequences.
type Style struct {
	FG            color.NRGBA
	BG            color.NRGBA
	FGSet         bool
	BGSet         bool
	Bold          bool
	Dim           bool
	Italic        bool
	Underline     bool
	Inverse       bool
	Strikethrough bool
}

// StyledChar is a single character with associated style.
type StyledChar struct {
	Char  rune
	Style Style
}

// StyledLine is a line of styled characters.
type StyledLine struct {
	Chars []StyledChar
}

// PlainText returns the line content as a plain string.
func (l *StyledLine) PlainText() string {
	runes := make([]rune, len(l.Chars))
	for i, c := range l.Chars {
		runes[i] = c.Char
	}
	return string(runes)
}

// ANSI standard 16-color palette.
var ANSIPalette = [16]color.NRGBA{
	{R: 0, G: 0, B: 0, A: 255},       // 0: Black
	{R: 170, G: 0, B: 0, A: 255},     // 1: Red
	{R: 0, G: 170, B: 0, A: 255},     // 2: Green
	{R: 170, G: 85, B: 0, A: 255},    // 3: Yellow/Brown
	{R: 0, G: 0, B: 170, A: 255},     // 4: Blue
	{R: 170, G: 0, B: 170, A: 255},   // 5: Magenta
	{R: 0, G: 170, B: 170, A: 255},   // 6: Cyan
	{R: 170, G: 170, B: 170, A: 255}, // 7: White
	{R: 85, G: 85, B: 85, A: 255},    // 8: Bright Black
	{R: 255, G: 85, B: 85, A: 255},   // 9: Bright Red
	{R: 85, G: 255, B: 85, A: 255},   // 10: Bright Green
	{R: 255, G: 255, B: 85, A: 255},  // 11: Bright Yellow
	{R: 85, G: 85, B: 255, A: 255},   // 12: Bright Blue
	{R: 255, G: 85, B: 255, A: 255},  // 13: Bright Magenta
	{R: 85, G: 255, B: 255, A: 255},  // 14: Bright Cyan
	{R: 255, G: 255, B: 255, A: 255}, // 15: Bright White
}

// ANSIColor returns a color from the 16-color ANSI palette.
func ANSIColor(index int) color.NRGBA {
	if index < 0 || index > 15 {
		return ANSIPalette[7] // default white
	}
	return ANSIPalette[index]
}

// Color256 returns a color from the 256-color palette.
func Color256(index int) color.NRGBA {
	if index < 16 {
		return ANSIColor(index)
	}
	if index < 232 {
		// 6x6x6 color cube
		index -= 16
		b := uint8((index % 6) * 51)
		g := uint8(((index / 6) % 6) * 51)
		r := uint8((index / 36) * 51)
		return color.NRGBA{R: r, G: g, B: b, A: 255}
	}
	// Grayscale ramp (232-255)
	level := uint8(8 + (index-232)*10)
	return color.NRGBA{R: level, G: level, B: level, A: 255}
}
