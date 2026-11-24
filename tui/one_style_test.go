package tui

import (
	"strings"
	"testing"

	"github.com/romaine-life/fzt/core"
)

// A list given without TreeMode draws as the menu does: the boxed search bar,
// not a bare prompt line (ATC-195: there is one style).
func TestEveryListDrawsAsTheMenu(t *testing.T) {
	row := func(name, about, target string) core.Item {
		return core.Item{Fields: []string{name, about}, ParentIdx: -1, PropertyOf: -1,
			Action: &core.ItemAction{Type: "command", Target: target}}
	}
	items := []core.Item{
		{Fields: []string{"Repo", "about"}, Depth: -1},
		row("at-claude", "the harness", "0"),
		row("fzt", "the engine", "1"),
	}
	cfg := Config{Layout: "reverse", Border: true, HeaderLines: 1, Title: "Start Claude in"}
	frames := Simulate(items, cfg, "fzt", 80, 12, false)
	if bar := strings.Split(frames[0].Content, "\n")[2]; !strings.Contains(bar, "type to search") {
		t.Fatalf("no search bar:\n%s", frames[0].Content)
	}
	if bar := strings.Split(frames[len(frames)-1].Content, "\n")[2]; !strings.Contains(bar, "fzt") {
		t.Fatalf("the search bar doesn't hold the search:\n%s", frames[len(frames)-1].Content)
	}
	s := NewTreeSession(items, cfg, 80, 12)
	if f := s.Render(); !strings.Contains(f.ANSI, "type to search") {
		t.Fatalf("session has no search bar:\n%s", f.ANSI)
	}
}
