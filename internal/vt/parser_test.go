package vt

import (
	"testing"
)

func TestPlainText(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("hello world"))
	if len(s.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(s.Lines))
	}
	got := s.Lines[0].PlainText()
	if got != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got)
	}
}

func TestNewline(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("line1\nline2\nline3"))
	if len(s.Lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(s.Lines))
	}
	if got := s.Lines[0].PlainText(); got != "line1" {
		t.Errorf("line 0: expected %q, got %q", "line1", got)
	}
	if got := s.Lines[1].PlainText(); got != "line2" {
		t.Errorf("line 1: expected %q, got %q", "line2", got)
	}
	if got := s.Lines[2].PlainText(); got != "line3" {
		t.Errorf("line 2: expected %q, got %q", "line3", got)
	}
}

func TestCarriageReturn(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("hello\rworld"))
	if len(s.Lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(s.Lines))
	}
	got := s.Lines[0].PlainText()
	if got != "world" {
		t.Errorf("expected %q, got %q", "world", got)
	}
}

func TestCRLF(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("line1\r\nline2"))
	if len(s.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(s.Lines))
	}
	if got := s.Lines[0].PlainText(); got != "line1" {
		t.Errorf("line 0: expected %q, got %q", "line1", got)
	}
	if got := s.Lines[1].PlainText(); got != "line2" {
		t.Errorf("line 1: expected %q, got %q", "line2", got)
	}
}

func TestSGRColors(t *testing.T) {
	s := NewScreen(80)
	// ESC[31m = red foreground, then text, then ESC[0m = reset
	s.Process([]byte("\x1b[31mred\x1b[0mnormal"))

	line := s.Lines[0]
	if len(line.Chars) != 9 { // "red" + "normal"
		t.Fatalf("expected 9 chars, got %d", len(line.Chars))
	}

	// First 3 chars should have red foreground
	for i := 0; i < 3; i++ {
		if !line.Chars[i].Style.FGSet {
			t.Errorf("char %d: expected FG to be set", i)
		}
		if line.Chars[i].Style.FG != ANSIColor(1) {
			t.Errorf("char %d: expected red, got %v", i, line.Chars[i].Style.FG)
		}
	}

	// Remaining chars should have default style
	for i := 3; i < 9; i++ {
		if line.Chars[i].Style.FGSet {
			t.Errorf("char %d: expected default FG", i)
		}
	}
}

func TestSGRBold(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("\x1b[1mbold\x1b[0mnormal"))

	line := s.Lines[0]
	for i := 0; i < 4; i++ {
		if !line.Chars[i].Style.Bold {
			t.Errorf("char %d should be bold", i)
		}
	}
	for i := 4; i < 10; i++ {
		if line.Chars[i].Style.Bold {
			t.Errorf("char %d should not be bold", i)
		}
	}
}

func TestTab(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("a\tb"))

	line := s.Lines[0]
	// 'a' at col 0, tab to col 8, 'b' at col 8
	if got := line.PlainText(); len(got) != 9 {
		t.Errorf("expected 9 chars (a + 7 spaces + b), got %d: %q", len(got), got)
	}
	if line.Chars[0].Char != 'a' {
		t.Errorf("expected 'a' at 0, got %c", line.Chars[0].Char)
	}
	if line.Chars[8].Char != 'b' {
		t.Errorf("expected 'b' at 8, got %c", line.Chars[8].Char)
	}
}

func TestEraseLine(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("hello world"))
	// Move cursor back 5, then erase to end of line
	s.Process([]byte("\x1b[5D\x1b[K"))
	got := s.Lines[0].PlainText()
	if got != "hello " {
		t.Errorf("expected %q, got %q", "hello ", got)
	}
}

func TestSnapshot(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("test"))

	snap := s.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("expected 1 line in snapshot, got %d", len(snap))
	}

	// Modify original
	s.Process([]byte("\nmore"))

	// Snapshot should be unchanged
	if len(snap) != 1 {
		t.Errorf("snapshot was modified")
	}
}

func TestUTF8(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("hello, world!"))
	got := s.Lines[0].PlainText()
	if got != "hello, world!" {
		t.Errorf("expected UTF-8 text, got %q", got)
	}
}

func Test256Color(t *testing.T) {
	s := NewScreen(80)
	// ESC[38;5;196m = 256-color foreground (red)
	s.Process([]byte("\x1b[38;5;196mtext\x1b[0m"))

	if !s.Lines[0].Chars[0].Style.FGSet {
		t.Error("expected FG to be set for 256-color")
	}
}

func TestTruecolor(t *testing.T) {
	s := NewScreen(80)
	// ESC[38;2;255;128;0m = truecolor foreground
	s.Process([]byte("\x1b[38;2;255;128;0mtext\x1b[0m"))

	ch := s.Lines[0].Chars[0]
	if !ch.Style.FGSet {
		t.Error("expected FG to be set for truecolor")
	}
	if ch.Style.FG.R != 255 || ch.Style.FG.G != 128 || ch.Style.FG.B != 0 {
		t.Errorf("expected (255,128,0), got (%d,%d,%d)", ch.Style.FG.R, ch.Style.FG.G, ch.Style.FG.B)
	}
}

func TestClearScreen(t *testing.T) {
	s := NewScreen(80)
	s.Process([]byte("hello\nworld"))
	s.Process([]byte("\x1b[2J"))

	// After clear, cursor should be at 0,0 and screen empty
	if s.curRow != 0 || s.curCol != 0 {
		t.Errorf("expected cursor at (0,0), got (%d,%d)", s.curRow, s.curCol)
	}
}
