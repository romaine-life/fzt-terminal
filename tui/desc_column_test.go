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

// The description column fits the entries of the folder you are in, the top
// level until one is entered, and holds still as the selection moves or a
// search opens folders in place; a longer name pushes only its own
// description unless it sits outside the folder you entered, where it is cut
// with an ellipsis (FT-14, FT-15).
func TestDescColumnFitsTheEnteredLevel(t *testing.T) {
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true}
	top := len("a-command-with-a-very-long-name") + 2 // the top level's longest name
	inNav := 2 + len("first-entry") + 2               // nav's longest entry, indented
	for _, tc := range []struct {
		name string
		keys string
		want map[string]int // a row's description, and where it should start
	}{
		{"top level, first row selected", "", map[string]int{
			"Show the links":  top,
			"Far away":        top,
			"Restart the app": top,
		}},
		{"top level, selection moved", "{down}{down}{down}", map[string]int{
			"Show the links":  top,
			"Far away":        top,
			"Restart the app": top,
		}},
		{"folder opened in place by a search", "entr{down}", map[string]int{
			"The first entry":  top,
			"The second entry": top,
			"Far away":         top,
		}},
		{"entered folder", "{down}{down}{enter}{down}", map[string]int{
			"The first entry":  inNav,
			"The second entry": inNav,
			"Restart the app":  inNav,
			"Far away":         inNav,
		}},
		{"entered folder, cursor outside it", "{down}{down}{enter}{up}{up}", map[string]int{
			"The second entry": inNav,
			"Show the links":   inNav,
		}},
		{"folder left again", "{down}{down}{enter}{esc}", map[string]int{
			"Show the links":  top,
			"Restart the app": top,
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
				"Far away":         {drawn("a-command-with-a-very-long-name", tc.keys), 0},
				"The first entry":  {"first-entry", 1},
				"The second entry": {"second", 1},
				"Restart the app":  {drawn("reload-the-app", tc.keys), 0},
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

// drawn is a top-level name as drawn: whole at the top level or in a folder
// only a search opened; once nav is entered, cut to nav's column less the gap
// (13 cells) with an ellipsis.
func drawn(name, keys string) string {
	if strings.Contains(keys, "{enter}") && !strings.HasSuffix(keys, "{esc}") && len(name) > 13 {
		return name[:12] + "…"
	}
	return name
}
