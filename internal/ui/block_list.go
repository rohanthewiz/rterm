package ui

import (
	"gioui.org/layout"

	"github.com/rohanthewiz/rterm/internal/model"
)

// BlockListView renders a scrollable list of blocks.
type BlockListView struct {
	list      layout.List
	blockView BlockView
}

// NewBlockListView creates a new block list.
func NewBlockListView() *BlockListView {
	return &BlockListView{
		list: layout.List{
			Axis:        layout.Vertical,
			ScrollToEnd: true,
		},
	}
}

// Layout renders the block list.
func (bl *BlockListView) Layout(gtx layout.Context, th *Theme, session *model.Session) layout.Dimensions {
	blocks := session.Blocks()
	return bl.list.Layout(gtx, len(blocks), func(gtx layout.Context, index int) layout.Dimensions {
		return bl.blockView.Layout(gtx, th, blocks[index])
	})
}
