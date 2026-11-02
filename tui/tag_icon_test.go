package tui

import (
	"strings"
	"testing"

	"github.com/romaine-life/fzt/core"
)

// Tag search shows as the search bar's icon: a tag in place of the magnifying
// glass, before and while typing, with the hint text unchanged (FZT-70).
func TestSearchBarIconShowsTagMode(t *testing.T) {
	items, err := core.LoadYAMLFromString(`
- name: tmux
  children:
    - name: new-session
      tags: [session, new]
`)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		tagMode   bool
		icon, not string
	}{
		{true, "", ""},
		{false, "", ""},
	} {
		cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, TagMode: tc.tagMode}
		frames := Simulate(items, cfg, "s", 60, 10, false)
		for i, f := range frames {
			p := promptLine(t, f.Content)
			if !strings.Contains(p, tc.icon) || strings.Contains(p, tc.not) {
				t.Fatalf("tag mode %v, frame %d: search box = %q, want icon %q\n%s", tc.tagMode, i, p, tc.icon, FormatFrames(frames))
			}
		}
		if p := promptLine(t, frames[0].Content); !strings.Contains(p, "type to search…") {
			t.Fatalf("tag mode %v: hint changed: %q", tc.tagMode, p)
		}
	}
}
