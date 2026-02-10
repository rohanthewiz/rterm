package app

import (
	"image"
	"os"

	gioapp "gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/rohanthewiz/rterm/internal/model"
	"github.com/rohanthewiz/rterm/internal/shell"
	"github.com/rohanthewiz/rterm/internal/ui"
)

// App is the top-level application.
type App struct {
	window    *gioapp.Window
	theme     *ui.Theme
	session   *model.Session
	engine    *shell.Engine
	editor    *ui.EditorWidget
	blockList *ui.BlockListView
}

// New creates a new App instance.
func New() *App {
	w := new(gioapp.Window)
	w.Option(
		gioapp.Title("rterm"),
		gioapp.Size(unit.Dp(900), unit.Dp(600)),
	)

	session := model.NewSession(80)
	th := ui.NewTheme()
	editor := ui.NewEditorWidget()
	blockList := ui.NewBlockListView()

	a := &App{
		window:    w,
		theme:     th,
		session:   session,
		editor:    editor,
		blockList: blockList,
	}

	a.engine = shell.NewEngine(session, w)

	return a
}

// Run starts the event loop. Call from a goroutine; app.Main() must be called
// on the main goroutine.
func (a *App) Run() error {
	var ops op.Ops

	for {
		switch e := a.window.Event().(type) {
		case gioapp.DestroyEvent:
			return e.Err

		case gioapp.FrameEvent:
			gtx := gioapp.NewContext(&ops, e)

			// Process editor submit
			if cmd, ok := a.editor.Update(gtx); ok && cmd != "" {
				a.engine.Execute(cmd)
			}

			// Layout
			a.layout(gtx)

			e.Frame(gtx.Ops)
		}
	}
}

func (a *App) layout(gtx layout.Context) layout.Dimensions {
	// Fill background
	defer clip.Rect(image.Rectangle{Max: gtx.Constraints.Max}).Push(gtx.Ops).Pop()
	paint.Fill(gtx.Ops, a.theme.BG)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		// Block list (takes remaining space)
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return a.blockList.Layout(gtx, a.theme, a.session)
		}),
		// Editor divider
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			size := image.Point{X: gtx.Constraints.Max.X, Y: gtx.Dp(unit.Dp(1))}
			defer clip.Rect(image.Rectangle{Max: size}).Push(gtx.Ops).Pop()
			paint.Fill(gtx.Ops, a.theme.DividerColor)
			return layout.Dimensions{Size: size}
		}),
		// Command editor (pinned at bottom)
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return a.editor.Layout(gtx, a.theme)
		}),
	)
}

// Window returns the underlying Gio window (used by main).
func (a *App) Window() *gioapp.Window {
	return a.window
}

// Main runs the platform event loop. Must be called from the main goroutine.
func Main() {
	gioapp.Main()
}

// Exit terminates the process.
func Exit(code int) {
	os.Exit(code)
}
