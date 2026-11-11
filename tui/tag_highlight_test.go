package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
)

type cellCanvas struct{ cells map[int]tcell.Style }

func (c *cellCanvas) SetContent(x, y int, r rune, comb []rune, st tcell.Style) { c.cells[x] = st }
func (c *cellCanvas) Size() (int, int)                                         { return 80, 1 }
func (c *cellCanvas) ShowCursor(x, y int)                                      {}
func (c *cellCanvas) HideCursor()                                              {}

// Every tag a word matched is highlighted, not only the one Tab completes to:
// rel matches relaunch and reload (FZT-80).
func TestDrawTagsHighlightsEveryMatchedTag(t *testing.T) {
	c := &cellCanvas{cells: map[int]tcell.Style{}}
	item := core.Item{Fields: []string{"claude-relaunch"}, Tags: []string{"claude", "relaunch", "reload"}}
	terms := []core.TagTerm{{Term: "rel", Tag: "reload", Tags: []string{"relaunch", "reload"}}}
	drawTags(c, 0, 0, item, terms, false, 80)
	hl := func(x int) bool { fg, _, _ := c.cells[x].Decompose(); return fg == HighlightFg }
	// "claude relaunch reload": claude at 0, relaunch at 7, reload at 16.
	if hl(0) || !hl(7) || !hl(16) {
		t.Fatalf("highlighted claude %v, relaunch %v, reload %v; want only relaunch and reload", hl(0), hl(7), hl(16))
	}
}
