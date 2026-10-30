package tui

import (
	"strings"
	"testing"
)

// With HidePromptBar the host draws the query; the frame starts at the rows,
// and a search still filters them.
func TestHidePromptBarStartsAtTheRows(t *testing.T) {
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, HidePromptBar: true}
	frames := Simulate(revealItems(8, 6), cfg, "move-tab", 60, 12, false)
	first := strings.Split(frames[0].Content, "\n")[0]
	if strings.Contains(frames[0].Content, "type to search") || !strings.HasSuffix(strings.TrimRight(first, " "), "cmd-00") {
		t.Fatalf("frame opens with a search box, not the rows:\n%s", frames[0].Content)
	}
	last := frames[len(frames)-1].Content
	if strings.Contains(last, "│ ") || !strings.Contains(last, "move-tab") {
		t.Fatalf("search frame:\n%s", last)
	}
}
