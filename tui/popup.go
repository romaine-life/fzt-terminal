package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
	"github.com/romaine-life/fzt/render"
)

// drawKeyPopup draws the `/` pop-up (core.OpenKeyPopup): the fzt underneath,
// drawn by under and faded, then the pop-up as an fzt of its own in the box
// core.KeyPopupBox places it in.
func drawKeyPopup(c render.Canvas, s *core.State, cfg Config, w, startY, h int, under func(render.Canvas)) {
	top := len(s.Contexts) - 1
	popup := s.Contexts[top : top+1]
	s.Contexts = s.Contexts[:top]
	mem := render.NewMemScreen(w, startY+h)
	under(mem)
	s.Contexts = s.Contexts[:top+1]

	for y := startY; y < startY+h; y++ {
		for x := 0; x < w; x++ {
			_, bg, _ := mem.Styles[y][x].Decompose()
			c.SetContent(x, y, mem.Grid[y][x], nil, tcell.StyleDefault.Foreground(UnderPopupFg).Background(bg))
		}
	}

	x, y, bw, bh := core.KeyPopupBox(s, cfg, w, h)
	box := inset{c: c, x: x, y: startY + y, w: bw, h: bh}
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			box.SetContent(bx, by, ' ', nil, tcell.StyleDefault)
		}
	}
	title := "/  what you can do here"
	if name := core.KeyPopupTitle(s); name != "" {
		title = "/  what you can do with " + name
	}
	view := &core.State{Contexts: popup, InspectTargetIdx: -1, EditTargetIdx: -1}
	c.HideCursor()
	drawTree(box, view, Config{Border: true, Tiered: cfg.Tiered, TreeMode: true, Title: title}, bw, 0, bh)
}

// inset is a canvas over a rectangle of another, so a whole fzt frame can be
// drawn inside it.
type inset struct {
	c          render.Canvas
	x, y, w, h int
}

func (o inset) SetContent(x, y int, r rune, comb []rune, st tcell.Style) {
	if x >= 0 && y >= 0 && x < o.w && y < o.h {
		o.c.SetContent(o.x+x, o.y+y, r, comb, st)
	}
}
func (o inset) Size() (int, int)    { return o.w, o.h }
func (o inset) ShowCursor(x, y int) { o.c.ShowCursor(o.x+x, o.y+y) }
func (o inset) HideCursor()         { o.c.HideCursor() }
