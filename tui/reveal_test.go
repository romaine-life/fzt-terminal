package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/romaine-life/fzt/core"
)

// listRows returns a frame's tree rows: the lines between the search box and
// the bottom of the screen, trailing blanks kept.
func listRows(t *testing.T, frame string) []string {
	t.Helper()
	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "└") {
			return lines[i+1:]
		}
	}
	t.Fatalf("no search box in frame:\n%s", frame)
	return nil
}

func rowOf(rows []string, name string) int {
	for i, row := range rows {
		if strings.Contains(row, " "+name+" ") || strings.HasSuffix(row, " "+name) {
			return i
		}
	}
	return -1
}

func revealItems(n int, folderAt int) []core.Item {
	var lines []string
	for i := 0; i < n; i++ {
		if i == folderAt {
			lines = append(lines, "0\tmove-tab", "1\tsession-a", "1\tsession-b")
			continue
		}
		lines = append(lines, fmt.Sprintf("0\tcmd-%02d", i))
	}
	items := core.ParseLines(lines, "\t", true, false)
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

// A match a search or Enter picked sits revealMargin rows below the search
// box with what follows it below, the end of the list left empty; a list
// that fits does not scroll (FZT-55).
func TestRevealPlacesTheMatch(t *testing.T) {
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true}
	for _, tc := range []struct {
		name     string
		n, at    int
		keys     string
		wantRow  int
		wantNext string // the row right after the match
	}{
		{"typed, mid-list", 60, 30, "move-tab", revealMargin, "cmd-31"},
		{"typed, end of list", 60, 59, "move-tab", revealMargin, ""},
		{"Enter opens it", 60, 30, "move-tab{enter}", revealMargin, "session-a"},
		{"Enter at end of list", 60, 59, "move-tab{enter}", revealMargin, "session-a"},
		{"list fits", 8, 6, "move-tab", 6, "cmd-07"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frames := Simulate(revealItems(tc.n, tc.at), cfg, tc.keys, 60, 20, false)
			rows := listRows(t, frames[len(frames)-1].Content)
			got := rowOf(rows, "move-tab")
			if got != tc.wantRow {
				t.Fatalf("move-tab on list row %d, want %d:\n%s", got, tc.wantRow, strings.Join(rows, "\n"))
			}
			next := strings.TrimSpace(rows[got+1])
			if tc.wantNext == "" {
				if next != "" {
					t.Fatalf("row after move-tab is %q, want empty (end of list)", next)
				}
			} else if !strings.HasSuffix(next, tc.wantNext) {
				t.Fatalf("row after move-tab is %q, want %s", next, tc.wantNext)
			}
		})
	}
}
