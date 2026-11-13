package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
)

// A tree row is drawn in the colours its input gave (a tmux window by its
// Claude state, TMUX-205), keeping the selection's background, with a search
// match highlighted over them; a row with none keeps the row's style.
func TestDrawRowTextKeepsInputColours(t *testing.T) {
	blue := tcell.NewRGBColor(57, 186, 230)
	styled := core.ParseANSI("\x1b[38;2;57;186;230mbuild\x1b[39m!")
	base := tcell.StyleDefault.Foreground(QueryFg).Background(SelectionBg)

	c := &cellCanvas{cells: map[int]tcell.Style{}}
	drawRowText(c, 0, 0, "build!", styled, base, 80, []int{0}, true)
	fgbg := func(x int) (tcell.Color, tcell.Color) { fg, bg, _ := c.cells[x].Decompose(); return fg, bg }
	if fg, bg := fgbg(1); fg != blue || bg != SelectionBg {
		t.Fatalf("a coloured character is %v on %v, want #39bae6 on the selection", fg, bg)
	}
	if fg, _ := fgbg(0); fg != HighlightFg {
		t.Fatalf("a matched character is %v, want the highlight", fg)
	}
	if fg, _ := fgbg(5); fg != QueryFg {
		t.Fatalf("a character the input left uncoloured is %v, want the row's %v", fg, QueryFg)
	}

	c = &cellCanvas{cells: map[int]tcell.Style{}}
	drawRowText(c, 0, 0, "plain", nil, tcell.StyleDefault, 80, nil, false)
	if c.cells[0] != tcell.StyleDefault {
		t.Fatalf("a row with no colours drew %v", c.cells[0])
	}
}
