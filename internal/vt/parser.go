package vt

import (
	"unicode/utf8"
)

type parserState int

const (
	stateGround parserState = iota
	stateEscape
	stateCSIParam
	stateOSCString
)

// Screen holds the terminal output state for a block.
type Screen struct {
	Lines    []*StyledLine
	curRow   int
	curCol   int
	curStyle Style
	width    int

	// Parser state
	state   parserState
	params  []int
	curP    int
	hasP    bool
	oscBuf  []byte
}

// NewScreen creates a screen with the given column width.
func NewScreen(width int) *Screen {
	return &Screen{
		width: width,
		Lines: []*StyledLine{{}},
	}
}

// Process feeds raw bytes from terminal output into the parser.
func (s *Screen) Process(data []byte) {
	for len(data) > 0 {
		if s.state == stateGround && data[0] >= 0x20 && data[0] != 0x7F {
			r, size := utf8.DecodeRune(data)
			if r != utf8.RuneError || size > 1 {
				s.putChar(r)
				data = data[size:]
				continue
			}
		}
		s.processByte(data[0])
		data = data[1:]
	}
}

func (s *Screen) processByte(b byte) {
	switch s.state {
	case stateGround:
		s.handleGround(b)
	case stateEscape:
		s.handleEscape(b)
	case stateCSIParam:
		s.handleCSIParam(b)
	case stateOSCString:
		s.handleOSCString(b)
	}
}

func (s *Screen) handleGround(b byte) {
	switch {
	case b == 0x1B: // ESC
		s.state = stateEscape
	case b == '\n': // LF
		s.newLine()
	case b == '\r': // CR
		s.curCol = 0
	case b == '\t': // TAB
		next := ((s.curCol / 8) + 1) * 8
		if s.width > 0 && next > s.width {
			next = s.width
		}
		for s.curCol < next {
			s.putChar(' ')
		}
	case b == 0x08: // BS (backspace)
		if s.curCol > 0 {
			s.curCol--
		}
	case b == 0x07: // BEL
		// Ignore bell
	case b >= 0x20 && b <= 0x7E: // Printable ASCII
		s.putChar(rune(b))
	}
}

func (s *Screen) handleEscape(b byte) {
	switch b {
	case '[': // CSI
		s.state = stateCSIParam
		s.params = s.params[:0]
		s.curP = 0
		s.hasP = false
	case ']': // OSC
		s.state = stateOSCString
		s.oscBuf = s.oscBuf[:0]
	case '(', ')': // Character set designation — ignore next byte
		s.state = stateGround
	default:
		// Other ESC sequences — ignore and return to ground
		s.state = stateGround
	}
}

func (s *Screen) handleCSIParam(b byte) {
	switch {
	case b >= '0' && b <= '9':
		s.curP = s.curP*10 + int(b-'0')
		s.hasP = true
	case b == ';':
		s.params = append(s.params, s.curP)
		s.curP = 0
		s.hasP = false
	case b >= 0x40 && b <= 0x7E: // Final byte
		if s.hasP || len(s.params) > 0 {
			s.params = append(s.params, s.curP)
		}
		s.dispatchCSI(b)
		s.state = stateGround
	default:
		// Intermediate bytes or unexpected — for Phase 1, just ignore
		if b >= 0x20 && b <= 0x2F {
			// Intermediate byte — skip
		} else {
			s.state = stateGround
		}
	}
}

func (s *Screen) handleOSCString(b byte) {
	switch b {
	case 0x07: // BEL terminates OSC
		s.state = stateGround
	case 0x1B: // ESC might start ST (ESC \)
		s.state = stateGround
	default:
		s.oscBuf = append(s.oscBuf, b)
	}
}

func (s *Screen) dispatchCSI(final byte) {
	switch final {
	case 'm': // SGR — Select Graphic Rendition
		s.handleSGR()
	case 'J': // ED — Erase in Display
		s.handleED()
	case 'K': // EL — Erase in Line
		s.handleEL()
	case 'A': // CUU — Cursor Up
		n := s.paramOrDefault(0, 1)
		s.curRow -= n
		if s.curRow < 0 {
			s.curRow = 0
		}
	case 'B': // CUD — Cursor Down
		n := s.paramOrDefault(0, 1)
		s.curRow += n
	case 'C': // CUF — Cursor Forward
		n := s.paramOrDefault(0, 1)
		s.curCol += n
	case 'D': // CUB — Cursor Back
		n := s.paramOrDefault(0, 1)
		s.curCol -= n
		if s.curCol < 0 {
			s.curCol = 0
		}
	case 'H', 'f': // CUP — Cursor Position
		row := s.paramOrDefault(0, 1)
		col := s.paramOrDefault(1, 1)
		s.curRow = row - 1
		s.curCol = col - 1
		if s.curRow < 0 {
			s.curRow = 0
		}
		if s.curCol < 0 {
			s.curCol = 0
		}
	case 'h', 'l': // SM/RM — Set/Reset Mode (ignore for Phase 1)
	case 'r': // DECSTBM — scrolling region (ignore)
	case 'c': // DA — Device Attributes (ignore)
	case 'n': // DSR — Device Status Report (ignore)
	}
}

func (s *Screen) paramOrDefault(index, defaultVal int) int {
	if index < len(s.params) && s.params[index] > 0 {
		return s.params[index]
	}
	return defaultVal
}

func (s *Screen) handleSGR() {
	if len(s.params) == 0 {
		s.curStyle = Style{}
		return
	}
	for i := 0; i < len(s.params); i++ {
		p := s.params[i]
		switch {
		case p == 0:
			s.curStyle = Style{}
		case p == 1:
			s.curStyle.Bold = true
		case p == 2:
			s.curStyle.Dim = true
		case p == 3:
			s.curStyle.Italic = true
		case p == 4:
			s.curStyle.Underline = true
		case p == 7:
			s.curStyle.Inverse = true
		case p == 9:
			s.curStyle.Strikethrough = true
		case p == 22:
			s.curStyle.Bold = false
			s.curStyle.Dim = false
		case p == 23:
			s.curStyle.Italic = false
		case p == 24:
			s.curStyle.Underline = false
		case p == 27:
			s.curStyle.Inverse = false
		case p == 29:
			s.curStyle.Strikethrough = false
		case p >= 30 && p <= 37:
			s.curStyle.FG = ANSIColor(p - 30)
			s.curStyle.FGSet = true
		case p == 38:
			i = s.parseExtendedColor(i, true)
		case p == 39:
			s.curStyle.FGSet = false
		case p >= 40 && p <= 47:
			s.curStyle.BG = ANSIColor(p - 40)
			s.curStyle.BGSet = true
		case p == 48:
			i = s.parseExtendedColor(i, false)
		case p == 49:
			s.curStyle.BGSet = false
		case p >= 90 && p <= 97:
			s.curStyle.FG = ANSIColor(p - 90 + 8)
			s.curStyle.FGSet = true
		case p >= 100 && p <= 107:
			s.curStyle.BG = ANSIColor(p - 100 + 8)
			s.curStyle.BGSet = true
		}
	}
}

func (s *Screen) parseExtendedColor(i int, fg bool) int {
	if i+1 >= len(s.params) {
		return i
	}
	switch s.params[i+1] {
	case 5: // 256-color
		if i+2 < len(s.params) {
			c := Color256(s.params[i+2])
			if fg {
				s.curStyle.FG = c
				s.curStyle.FGSet = true
			} else {
				s.curStyle.BG = c
				s.curStyle.BGSet = true
			}
			return i + 2
		}
	case 2: // Truecolor
		if i+4 < len(s.params) {
			r := uint8(s.params[i+2])
			g := uint8(s.params[i+3])
			b := uint8(s.params[i+4])
			c := ANSIPalette[0]
			c.R, c.G, c.B = r, g, b
			if fg {
				s.curStyle.FG = c
				s.curStyle.FGSet = true
			} else {
				s.curStyle.BG = c
				s.curStyle.BGSet = true
			}
			return i + 4
		}
	}
	return i + 1
}

func (s *Screen) handleED() {
	p := s.paramOrDefault(0, 0)
	switch p {
	case 0: // Clear from cursor to end
		s.clearLineFrom(s.curRow, s.curCol)
		for i := s.curRow + 1; i < len(s.Lines); i++ {
			s.Lines[i] = &StyledLine{}
		}
	case 1: // Clear from start to cursor
		for i := 0; i < s.curRow; i++ {
			s.Lines[i] = &StyledLine{}
		}
		s.clearLineTo(s.curRow, s.curCol)
	case 2: // Clear entire screen
		s.Lines = []*StyledLine{{}}
		s.curRow = 0
		s.curCol = 0
	}
}

func (s *Screen) handleEL() {
	p := s.paramOrDefault(0, 0)
	switch p {
	case 0: // Clear from cursor to end of line
		s.clearLineFrom(s.curRow, s.curCol)
	case 1: // Clear from start to cursor
		s.clearLineTo(s.curRow, s.curCol)
	case 2: // Clear entire line
		if s.curRow < len(s.Lines) {
			s.Lines[s.curRow] = &StyledLine{}
		}
	}
}

func (s *Screen) clearLineFrom(row, col int) {
	if row >= len(s.Lines) {
		return
	}
	line := s.Lines[row]
	if col < len(line.Chars) {
		line.Chars = line.Chars[:col]
	}
}

func (s *Screen) clearLineTo(row, col int) {
	if row >= len(s.Lines) {
		return
	}
	line := s.Lines[row]
	for i := 0; i < col && i < len(line.Chars); i++ {
		line.Chars[i] = StyledChar{Char: ' '}
	}
}

func (s *Screen) ensureLine(row int) {
	for len(s.Lines) <= row {
		s.Lines = append(s.Lines, &StyledLine{})
	}
}

func (s *Screen) putChar(r rune) {
	s.ensureLine(s.curRow)
	line := s.Lines[s.curRow]

	// Extend line if needed
	for len(line.Chars) <= s.curCol {
		line.Chars = append(line.Chars, StyledChar{Char: ' '})
	}

	line.Chars[s.curCol] = StyledChar{Char: r, Style: s.curStyle}
	s.curCol++

	// Wrap at width
	if s.width > 0 && s.curCol >= s.width {
		s.curCol = 0
		s.curRow++
		s.ensureLine(s.curRow)
	}
}

func (s *Screen) newLine() {
	s.curRow++
	s.curCol = 0
	s.ensureLine(s.curRow)
}

// Snapshot returns a copy of the current lines for thread-safe reading.
func (s *Screen) Snapshot() []StyledLine {
	result := make([]StyledLine, len(s.Lines))
	for i, line := range s.Lines {
		if line != nil {
			chars := make([]StyledChar, len(line.Chars))
			copy(chars, line.Chars)
			result[i] = StyledLine{Chars: chars}
		}
	}
	return result
}
