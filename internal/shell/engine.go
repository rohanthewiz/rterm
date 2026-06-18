package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/creack/pty"
	"github.com/rohanthewiz/rterm/internal/model"
)

// Invalidator is called to request a UI re-render.
type Invalidator interface {
	Invalidate()
}

// Engine handles command execution.
type Engine struct {
	Session *model.Session
	CWD     string
	inv     Invalidator

	mu         sync.Mutex
	rows, cols int
}

// NewEngine creates a new shell engine.
func NewEngine(session *model.Session, inv Invalidator) *Engine {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}
	return &Engine{
		Session: session,
		CWD:     cwd,
		inv:     inv,
		rows:    24,
		cols:    80,
	}
}

// SetSize updates the terminal dimensions for new commands and resizes any
// currently-running command's pty (which delivers SIGWINCH to the child).
func (e *Engine) SetSize(rows, cols int) {
	if rows <= 0 || cols <= 0 {
		return
	}
	e.mu.Lock()
	if rows == e.rows && cols == e.cols {
		e.mu.Unlock()
		return
	}
	e.rows, e.cols = rows, cols
	e.mu.Unlock()

	e.Session.SetCols(cols)
	for _, b := range e.Session.Blocks() {
		if !b.Done() {
			b.Resize(rows, cols)
		}
	}
}

func (e *Engine) size() (rows, cols int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.rows, e.cols
}

// InterruptLatest sends ^C to the most recently started running command.
// Returns true if a running command was found.
func (e *Engine) InterruptLatest() bool {
	blocks := e.Session.Blocks()
	for i := len(blocks) - 1; i >= 0; i-- {
		if !blocks[i].Done() {
			_ = blocks[i].Interrupt()
			return true
		}
	}
	return false
}

// Execute runs a command asynchronously and returns the block tracking it.
func (e *Engine) Execute(command string) *model.Block {
	trimmed := strings.TrimSpace(command)

	// Handle cd as a built-in
	if trimmed == "cd" || strings.HasPrefix(trimmed, "cd ") {
		return e.handleCD(command, trimmed)
	}

	block := e.Session.AddBlock(command, e.CWD)
	go e.runCommand(block, trimmed)
	return block
}

func (e *Engine) handleCD(raw, trimmed string) *model.Block {
	block := e.Session.AddBlock(raw, e.CWD)

	target := strings.TrimSpace(strings.TrimPrefix(trimmed, "cd"))
	if target == "" || target == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			block.AppendOutput([]byte(fmt.Sprintf("cd: %v\n", err)))
			block.Finish(1)
			e.inv.Invalidate()
			return block
		}
		target = home
	} else if strings.HasPrefix(target, "~/") {
		home, _ := os.UserHomeDir()
		target = filepath.Join(home, target[2:])
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(e.CWD, target)
	}
	target = filepath.Clean(target)

	info, err := os.Stat(target)
	if err != nil {
		block.AppendOutput([]byte(fmt.Sprintf("cd: %v\n", err)))
		block.Finish(1)
		e.inv.Invalidate()
		return block
	}
	if !info.IsDir() {
		block.AppendOutput([]byte(fmt.Sprintf("cd: %s: Not a directory\n", target)))
		block.Finish(1)
		e.inv.Invalidate()
		return block
	}

	e.CWD = target
	block.Finish(0)
	e.inv.Invalidate()
	return block
}

func (e *Engine) runCommand(block *model.Block, command string) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = e.CWD
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		block.AppendOutput([]byte(fmt.Sprintf("error: %v\n", err)))
		block.Finish(127)
		e.inv.Invalidate()
		return
	}
	defer ptmx.Close()

	rows, cols := e.size()
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})

	// Let the UI interrupt (^C) and resize this command while it runs.
	block.SetController(
		func() error {
			_, err := ptmx.Write([]byte{0x03})
			return err
		},
		func(r, c int) error {
			return pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(r), Cols: uint16(c)})
		},
	)

	buf := make([]byte, 4096)
	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			block.AppendOutput(buf[:n])
			e.inv.Invalidate()
		}
		if err != nil {
			break
		}
	}

	_ = cmd.Wait()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	block.Finish(exitCode)
	e.inv.Invalidate()
}
