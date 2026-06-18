package model

import "testing"

func TestBlockInterruptClearedAfterFinish(t *testing.T) {
	b := NewBlock(1, "sleep 10", "/", 80)

	interrupts := 0
	b.SetController(func() error { interrupts++; return nil }, nil)

	if err := b.Interrupt(); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}
	if interrupts != 1 {
		t.Fatalf("expected 1 interrupt, got %d", interrupts)
	}

	// After finishing, the controller is dropped and Interrupt is a no-op.
	b.Finish(0)
	if err := b.Interrupt(); err != nil {
		t.Fatalf("Interrupt after finish: %v", err)
	}
	if interrupts != 1 {
		t.Fatalf("expected interrupt to be a no-op after finish, got %d calls", interrupts)
	}
}

func TestBlockResizeInvokesController(t *testing.T) {
	b := NewBlock(1, "top", "/", 80)

	var gotRows, gotCols int
	b.SetController(nil, func(r, c int) error { gotRows, gotCols = r, c; return nil })

	b.Resize(30, 120)
	if gotRows != 30 || gotCols != 120 {
		t.Fatalf("expected resize(30,120), got resize(%d,%d)", gotRows, gotCols)
	}
}

func TestSessionSetColsAppliesToNewBlocks(t *testing.T) {
	s := NewSession(80)
	s.SetCols(10)

	b := s.AddBlock("echo", "/")
	b.AppendOutput([]byte("aaaaaaaaaaaa")) // 12 chars wraps at width 10

	lines := b.OutputLines()
	nonEmpty := 0
	for _, l := range lines {
		if len(l.Chars) > 0 {
			nonEmpty++
		}
	}
	if nonEmpty < 2 {
		t.Fatalf("expected output to wrap into >=2 lines at width 10, got %d", nonEmpty)
	}
}
