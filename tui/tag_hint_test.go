package tui

import (
	"strings"
	"testing"

	"github.com/romaine-life/fzt/core"
)

// tagHintItems is a menu whose tmux folder holds tagged commands and whose
// notes folder holds untagged ones.
func tagHintItems(t *testing.T) []core.Item {
	t.Helper()
	items, err := core.LoadYAMLFromString(`
- name: tmux
  children:
    - name: new-session
      tags: [session, new]
    - name: kill-window
      tags: [window, kill]
    - name: layouts
      children:
        - name: tiled
          tags: [layout]
- name: notes
  children:
    - name: groceries
    - name: todo
`)
	if err != nil {
		t.Fatal(err)
	}
	return items
}

// While tag search is on, the empty search bar says so before anything is
// typed (FZT-70), at the root, in a folder, and after a search is cleared.
func TestSearchBarHintShowsTagMode(t *testing.T) {
	base := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true}
	for _, tc := range []struct {
		name  string
		cfg   func(Config) Config
		keys  string
		frame int
		want  string
	}{
		{"root, tag mode on", func(c Config) Config { c.TagMode = true; return c }, "", 0, "search by tag…"},
		{"root, tag mode off", func(c Config) Config { return c }, "", 0, "type to search…"},
		{"tagged folder", func(c Config) Config { c.TagMode = true; c.StartFolder = []string{"tmux"}; return c }, "", 0, "search by tag…"},
		{"after a cleared search", func(c Config) Config { c.TagMode = true; return c }, "s{bs}", 2, "search by tag…"},
		{"untagged folder searches by name", func(c Config) Config { c.TagMode = true; c.StartFolder = []string{"notes"}; return c }, "", 0, "search…"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frames := Simulate(tagHintItems(t), tc.cfg(base), tc.keys, 70, 14, false)
			p := promptLine(t, frames[tc.frame].Content)
			got := strings.TrimSpace(strings.Trim(p, "│ "))
			if !strings.HasSuffix(got, tc.want) || (tc.want != "search by tag…" && strings.Contains(got, "tag")) {
				t.Fatalf("search box = %q, want it to end %q\n%s", p, tc.want, FormatFrames(frames[tc.frame:tc.frame+1]))
			}
		})
	}
}

// The folder picker's hint names tag search the same way.
func TestFolderPickerHintShowsTagMode(t *testing.T) {
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true,
		ScopedView: true, FoldersOnly: true, StartFolder: []string{"tmux"}}
	for _, tc := range []struct {
		tagMode bool
		want    string
	}{
		{true, "Enter picks this folder, Right opens one, or search by tag…"},
		{false, "Enter picks this folder, Right opens one, or type to search…"},
	} {
		cfg.TagMode = tc.tagMode
		frames := Simulate(tagHintItems(t), cfg, "", 100, 14, false)
		if p := promptLine(t, frames[0].Content); !strings.Contains(p, tc.want) {
			t.Fatalf("tag mode %v: search box = %q, want %q\n%s", tc.tagMode, p, tc.want, FormatFrames(frames))
		}
	}
}
