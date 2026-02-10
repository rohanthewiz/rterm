package model

import (
	"sync"
	"sync/atomic"
)

// Session holds the list of blocks for the current terminal session.
type Session struct {
	mu     sync.Mutex
	blocks []*Block
	nextID atomic.Uint64
	cols   int
}

// NewSession creates a new session with the given terminal width.
func NewSession(cols int) *Session {
	return &Session{cols: cols}
}

// AddBlock creates a new block, appends it to the session, and returns it.
func (s *Session) AddBlock(command, cwd string) *Block {
	id := s.nextID.Add(1)
	b := NewBlock(id, command, cwd, s.cols)
	s.mu.Lock()
	s.blocks = append(s.blocks, b)
	s.mu.Unlock()
	return b
}

// Blocks returns a snapshot of all blocks.
func (s *Session) Blocks() []*Block {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Block, len(s.blocks))
	copy(out, s.blocks)
	return out
}

// Len returns the number of blocks.
func (s *Session) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.blocks)
}
