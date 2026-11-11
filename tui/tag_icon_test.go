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

// Inside a TagDefault folder the search bar shows the tag icon with tag mode
// off, and backing out to the root shows the magnifying glass again (TMUX-196:
// Ctrl+B opens the at menu inside its tmux folder, which searches by tag).
func TestSearchBarIconShowsTagDefaultFolder(t *testing.T) {
	items, err := core.LoadYAMLFromString(`
- name: tmux
  children:
    - name: new-session
      tags: [session, new]
- name: lights
`)
	if err != nil {
		t.Fatal(err)
	}
	items[0].TagDefault = true
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, StartFolder: []string{"tmux"}}
	frames := Simulate(items, cfg, "{esc}", 60, 10, false)
	if p := promptLine(t, frames[0].Content); !strings.Contains(p, "") || strings.Contains(p, "") {
		t.Fatalf("inside the tmux folder: search box = %q, want the tag icon\n%s", p, FormatFrames(frames))
	}
	if p := promptLine(t, frames[len(frames)-1].Content); !strings.Contains(p, "") || strings.Contains(p, "") {
		t.Fatalf("back at the root: search box = %q, want the magnifying glass\n%s", p, FormatFrames(frames))
	}
}
