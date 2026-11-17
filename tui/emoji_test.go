package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
)

func emojiItems() []core.Item {
	items := core.ParseLines([]string{
		"👍🏽 thumbs up\tmedium skin tone",
		"👨‍👩‍👧 family\tman, woman, girl",
		"😀 grinning\tface",
	}, "\t", false, false)
	for i := range items {
		items[i].ParentIdx = -1
	}
	return items
}

// Each emoji draws whole, in the columns the terminal gives it, so the
// descriptions line up (GVT-12).
func TestEmojiRowsDrawWholeAndAligned(t *testing.T) {
	sess := NewTreeSession(emojiItems(), Config{Layout: "reverse", Border: true, Tiered: true, TreeMode: true, HidePalette: true, Title: "Emoji 😀"}, 60, 10)
	sess.Render()
	rows := strings.Split(sess.Screen().Snapshot(), "\n")
	descCol := -1
	for _, want := range [][2]string{{"👍🏽 thumbs up", "medium skin tone"}, {"👨‍👩‍👧 family", "man, woman, girl"}, {"😀 grinning", "face"}} {
		found := false
		for _, row := range rows {
			n, d := strings.Index(row, want[0]), strings.Index(row, want[1])
			if n < 0 || d < 0 {
				continue
			}
			found = true
			col := core.Width(row[:d])
			if descCol >= 0 && col != descCol {
				t.Errorf("%q starts at column %d, the others at %d:\n%s", want[1], col, descCol, strings.Join(rows, "\n"))
			}
			descCol = col
			if core.Width(row) != 60 {
				t.Errorf("row %q is %d columns wide, want 60", row, core.Width(row))
			}
		}
		if !found {
			t.Errorf("no row with %q then %q:\n%s", want[0], want[1], strings.Join(rows, "\n"))
		}
	}
	if !strings.Contains(rows[0], " Emoji 😀 ") {
		t.Errorf("title row %q", rows[0])
	}
}

// The search box's caret follows the columns an emoji takes.
func TestQueryCaretCountsEmojiColumns(t *testing.T) {
	sess := NewTreeSession(emojiItems(), Config{Layout: "reverse", Border: true, Tiered: true, TreeMode: true, HidePalette: true}, 60, 10)
	sess.HandleKeyResult(tcell.KeyRune, 'a', false)
	before := sess.Render().CursorX
	sess.HandleKeyResult(tcell.KeyRune, '😀', false)
	if got := sess.Render().CursorX; got != before+2 {
		t.Fatalf("caret moved from %d to %d after an emoji, want %d", before, got, before+2)
	}
}
