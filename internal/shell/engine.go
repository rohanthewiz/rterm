package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	}
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

	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

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
