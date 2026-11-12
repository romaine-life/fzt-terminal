package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
	"github.com/romaine-life/fzt/render"
)

func popupMenu(t *testing.T) []core.Item {
	t.Helper()
	items, err := core.LoadYAMLFromString(`
- name: workspace
  description: Manage workspace entries
  children:
    - name: wsopen
      description: Open a workspace in VS Code
      action: wsopen
    - name: wsload
      action: wsload
- name: lights
  action: lights
`)
	if err != nil {
		t.Fatal(err)
	}
	return items
}

// `/` draws the pop-up in its own box over the menu: titled with what it
// acts on, its rows with keycaps, and the menu's search still in sight.
func TestSlashDrawsThePopupOverTheMenu(t *testing.T) {
	cfg := Config{Layout: "reverse", Border: true, Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, Nth: []int{1}}
	frames := Simulate(popupMenu(t), cfg, "wsopen/", 100, 30, false)
	last := frames[len(frames)-1].Content
	for _, want := range []string{"wsopen", "/  what you can do with wsopen", "Run wsopen", " Enter ", " Shift+Enter ", "type to search"} {
		if !strings.Contains(last, want) {
			t.Fatalf("pop-up frame lacks %q:\n%s", want, last)
		}
	}
	if !strings.Contains(strings.Split(last, "\n")[2], "wsopen") {
		t.Fatalf("the menu's search isn't in sight above the pop-up:\n%s", last)
	}
}

// The menu under the pop-up is faded; the pop-up's own rows are not.
func TestMenuUnderThePopupIsFaded(t *testing.T) {
	cfg := Config{Layout: "reverse", Border: true, Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, Nth: []int{1}}
	s, cols := core.NewState(popupMenu(t), cfg)
	ctx := s.TopCtx()
	ctx.TreeExpanded, ctx.QueryExpanded, ctx.TreeCursor = map[int]bool{}, map[int]bool{}, -1
	for _, r := range "wsopen/" {
		core.HandleUnifiedKeyResult(s, tcell.KeyRune, r, false, cfg, cols)
	}
	mem := render.NewMemScreen(100, 30)
	drawUnified(mem, s, cfg, 100, 0, 30)
	fg, _, _ := mem.Styles[2][3].Decompose() // the menu's query
	if fg != UnderPopupFg {
		t.Fatalf("the menu's query isn't faded: fg %v", fg)
	}
	_, y, _, _ := core.KeyPopupBox(s, cfg, 100, 30)
	row := []rune(strings.Split(mem.Snapshot(), "\n")[y+4])
	at := strings.Index(string(row), "run")
	if at < 0 {
		t.Fatalf("no run row at the box's first row: %q", string(row))
	}
	if fg, _, _ := mem.Styles[y+4][len([]rune(string(row)[:at]))].Decompose(); fg == UnderPopupFg {
		t.Fatal("the pop-up's own rows are faded")
	}
}

// A flat list (the fzt CLI's picker, Tab completion) gets the pop-up too.
func TestSlashDrawsThePopupOverAFlatList(t *testing.T) {
	cfg := Config{Nth: []int{1}}
	items := core.ParseLines([]string{"wsopen", "wsload", "lights"}, "\t", false, false)
	frames := Simulate(items, cfg, "wsl/", 80, 24, false)
	last := frames[len(frames)-1].Content
	if !strings.Contains(last, "/  what you can do with wsload") || !strings.Contains(last, "Run wsload") {
		t.Fatalf("flat pop-up frame:\n%s", last)
	}
}
