package tui

import (
	"strings"
	"testing"

	"github.com/romaine-life/fzt/core"
)

// descColumnItems is a menu whose longest name sits outside the folder the
// tests open, so a column fitted to the whole menu would show.
func descColumnItems() []core.Item {
	items := core.ParseLines([]string{
		"0\tlinks\tShow the links",
		"0\ta-command-with-a-very-long-name\tFar away",
		"0\tnav\tGo somewhere",
		"1\tfirst-entry\tThe first entry",
		"1\tsecond\tThe second entry",
		"0\treload-the-app\tRestart the app",
	}, "\t", true, false)
	var stack []int
	for i := range items {
		for len(stack) > 0 && items[stack[len(stack)-1]].Depth >= items[i].Depth {
			stack = stack[:len(stack)-1]
		}
		items[i].ParentIdx = -1
		if len(stack) > 0 {
			p := stack[len(stack)-1]
			items[i].ParentIdx = p
			items[p].HasChildren = true
			items[p].Children = append(items[p].Children, i)
		}
		stack = append(stack, i)
	}
	return items
}

// descAt is where a row's description starts, counted as descColumn counts:
// the row's indent plus the cells from its name to its description.
func descAt(t *testing.T, rows []string, name, desc string, depth int) int {
	t.Helper()
	for _, row := range rows {
		n := strings.Index(row, " "+name+" ")
		d := strings.Index(row, desc)
		if n < 0 || d < 0 {
			continue
		}
		return depth*2 + len([]rune(row[:d])) - len([]rune(row[:n+1]))
	}
	t.Fatalf("no row %q with %q in:\n%s", name, desc, strings.Join(rows, "\n"))
	return 0
}

// The description column fits the selected row until a folder is entered,
// then that folder's entries; a longer name pushes only its own description
// (FT-7).
func TestDescColumnFollowsTheSelection(t *testing.T) {
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true}
	long := len("a-command-with-a-very-long-name") + 1
	for _, tc := range []struct {
		name string
		keys string
		want map[string]int // a row's description, and where it should start
	}{
		{"selected row", "{down}{down}{down}{down}", map[string]int{
			"Restart the app": len("reload-the-app") + 2,
			"Show the links":  len("reload-the-app") + 2,
			"Far away":        long,
		}},
		{"short selected row keeps the minimum", "{down}", map[string]int{
			"Show the links": 12,
			"Far away":       long,
		}},
		{"entered folder", "{down}{down}{down}{enter}{down}", map[string]int{
			"The first entry":  2 + len("first-entry") + 2,
			"The second entry": 2 + len("first-entry") + 2,
			"Restart the app":  2 + len("first-entry") + 2,
			"Far away":         long,
		}},
		{"entered folder, cursor outside it", "{down}{down}{down}{enter}{up}{up}", map[string]int{
			"The second entry": 2 + len("first-entry") + 2,
			"Show the links":   2 + len("first-entry") + 2,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frames := Simulate(descColumnItems(), cfg, tc.keys, 80, 14, false)
			rows := listRows(t, frames[len(frames)-1].Content)
			names := map[string]struct {
				name  string
				depth int
			}{
				"Show the links":   {"links", 0},
				"Far away":         {"a-command-with-a-very-long-name", 0},
				"The first entry":  {"first-entry", 1},
				"The second entry": {"second", 1},
				"Restart the app":  {"reload-the-app", 0},
			}
			for desc, want := range tc.want {
				row := names[desc]
				if got := descAt(t, rows, row.name, desc, row.depth); got != want {
					t.Errorf("%q starts at %d, want %d:\n%s", desc, got, want, strings.Join(rows, "\n"))
				}
			}
		})
	}
}
