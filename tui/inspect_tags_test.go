package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
)

func TestItemEditorShowsAndEditsTags(t *testing.T) {
	items, err := core.LoadYAMLFromString(`
- name: lightson
  action: lightson
  tags: [lights, home]
`)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Tiered: true, TreeMode: true}
	s, searchCols := core.NewState(items, cfg)
	ctx := s.TopCtx()
	ctx.TreeExpanded = map[int]bool{}
	ctx.QueryExpanded = map[int]bool{}
	ctx.TreeCursor = 0

	s.EditMode = "inspect"
	handleShiftEnter(s)
	var tagsRow = -1
	for vi, row := range core.TreeVisibleItems(s) {
		if row.Item.IsProperty && row.Item.PropertyKey == "tags" {
			tagsRow = vi
			if row.Item.Fields[1] != "lights home" {
				t.Fatalf("tags row shows %q", row.Item.Fields[1])
			}
		}
	}
	if tagsRow < 0 {
		t.Fatal("the item editor has no tags row")
	}

	ctx.TreeCursor = tagsRow
	handleShiftEnter(s)
	s.EditBuffer = []rune("lights, night")
	core.HandleUnifiedKey(s, tcell.KeyEnter, 0, false, cfg, searchCols)
	if got := core.FormatTags(ctx.AllItems[0].Tags); got != "lights night" {
		t.Fatalf("tags after edit = %q", got)
	}
}
