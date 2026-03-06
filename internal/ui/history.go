package ui

// History provides simple in-memory command history with navigation.
type History struct {
	entries []string
	pos     int  // current navigation position; len(entries) means "at bottom"
	saved   string // text saved when user starts navigating up
}

// Add records a command in history.
func (h *History) Add(cmd string) {
	if cmd == "" {
		return
	}
	// Deduplicate consecutive entries
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == cmd {
		h.pos = len(h.entries)
		return
	}
	h.entries = append(h.entries, cmd)
	h.pos = len(h.entries)
}

// Previous returns the previous command. currentText is saved on first
// navigation so it can be restored with Next.
func (h *History) Previous(currentText string) (string, bool) {
	if len(h.entries) == 0 || h.pos <= 0 {
		return "", false
	}
	if h.pos == len(h.entries) {
		h.saved = currentText
	}
	h.pos--
	return h.entries[h.pos], true
}

// Next returns the next command, or the saved text if at the bottom.
func (h *History) Next() (string, bool) {
	if h.pos >= len(h.entries) {
		return "", false
	}
	h.pos++
	if h.pos == len(h.entries) {
		return h.saved, true
	}
	return h.entries[h.pos], true
}

// Entries returns all history entries for searching.
func (h *History) Entries() []string {
	out := make([]string, len(h.entries))
	copy(out, h.entries)
	return out
}
