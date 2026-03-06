package ui

import (
	"gioui.org/layout"

	"github.com/rohanthewiz/rterm/internal/model"
)

// BlockListView renders a scrollable list of blocks with per-block state.
type BlockListView struct {
	list       layout.List
	blockView  BlockView
	states     map[uint64]*blockState
	searchTerm string

	// Actions collected during a single frame
	PendingActions []BlockResult
}

// NewBlockListView creates a new block list.
func NewBlockListView() *BlockListView {
	return &BlockListView{
		list: layout.List{
			Axis:        layout.Vertical,
			ScrollToEnd: true,
		},
		states: make(map[uint64]*blockState),
	}
}

// SetSearchTerm sets the current search term for highlighting.
func (bl *BlockListView) SetSearchTerm(term string) {
	bl.searchTerm = term
}

// CollapseAll collapses all blocks.
func (bl *BlockListView) CollapseAll() {
	for _, s := range bl.states {
		s.collapsed = true
	}
}

// ExpandAll expands all blocks.
func (bl *BlockListView) ExpandAll() {
	for _, s := range bl.states {
		s.collapsed = false
	}
}

// Layout renders the block list and collects actions.
func (bl *BlockListView) Layout(gtx layout.Context, th *Theme, session *model.Session) layout.Dimensions {
	blocks := session.Blocks()
	bl.PendingActions = bl.PendingActions[:0]

	return bl.list.Layout(gtx, len(blocks), func(gtx layout.Context, index int) layout.Dimensions {
		block := blocks[index]

		// Get or create per-block state
		state, ok := bl.states[block.ID]
		if !ok {
			state = &blockState{}
			bl.states[block.ID] = state
		}

		dims, result := bl.blockView.Layout(gtx, th, block, state, bl.searchTerm)
		if result.Action != BlockActionNone {
			bl.PendingActions = append(bl.PendingActions, result)
		}
		return dims
	})
}
