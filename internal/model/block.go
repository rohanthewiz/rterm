package model

import (
	"sync"
	"time"

	"github.com/rohanthewiz/rterm/internal/vt"
)

// Block represents a single command execution and its output.
type Block struct {
	ID        uint64
	Command   string
	StartTime time.Time
	EndTime   time.Time
	ExitCode  int    // -1 while running
	CWD       string // working directory at execution time

	mu     sync.Mutex
	screen *vt.Screen
	done   bool
}

// NewBlock creates a new block for the given command.
func NewBlock(id uint64, command, cwd string, cols int) *Block {
	return &Block{
		ID:        id,
		Command:   command,
		CWD:       cwd,
		StartTime: time.Now(),
		ExitCode:  -1,
		screen:    vt.NewScreen(cols),
	}
}

// AppendOutput feeds raw terminal output bytes into this block.
func (b *Block) AppendOutput(data []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.screen.Process(data)
}

// Finish marks the block as complete with the given exit code.
func (b *Block) Finish(exitCode int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ExitCode = exitCode
	b.EndTime = time.Now()
	b.done = true
}

// Done reports whether the command has finished.
func (b *Block) Done() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.done
}

// OutputLines returns a snapshot of the current output for rendering.
func (b *Block) OutputLines() []vt.StyledLine {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.screen.Snapshot()
}
