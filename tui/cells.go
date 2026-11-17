package tui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/uniseg"
	"github.com/romaine-life/fzt/core"
	"github.com/romaine-life/fzt/render"
)

// drawClusters draws text from column x one grapheme cluster per cell, each
// as wide as the terminal shows it (an emoji takes two columns, a skin-toned
// or joined one stays whole), while it fits in maxW columns. style gets the
// index of the cluster's first rune, since match indices and ANSI styles
// count runes. It returns the columns drawn.
func drawClusters(c render.Canvas, x, y int, text string, maxW int, style func(i int) tcell.Style) int {
	col, i, state := 0, 0, -1
	for text != "" {
		var cluster string
		cluster, text, _, state = uniseg.FirstGraphemeClusterInString(text, state)
		runes := []rune(cluster)
		w := max(core.Width(cluster), 1)
		if col+w > maxW {
			break
		}
		c.SetContent(x+col, y, runes[0], runes[1:], style(i))
		col += w
		i += len(runes)
	}
	return col
}

// queryStart is the first rune of query to draw so the cursor (a rune index)
// stays inside a box w columns wide, starting on a cluster boundary.
func queryStart(query []rune, cursor, w int) int {
	start, state := 0, -1
	rest := string(query)
	for start < cursor && core.Width(string(query[start:cursor])) > w-1 {
		var cluster string
		cluster, rest, _, state = uniseg.FirstGraphemeClusterInString(rest, state)
		start += len([]rune(cluster))
	}
	return start
}
