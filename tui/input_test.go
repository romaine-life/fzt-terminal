package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
)

// pressKeys runs keys through the same steps as runWithSession and returns
// the pick that ended the picker, or a zero Pick if none did.
func pressKeys(t *testing.T, s *core.State, cfg Config, searchCols []int, keys ...any) Pick {
	t.Helper()
	var inputItem core.Item
	for _, k := range keys {
		key, ch := tcell.KeyRune, rune(0)
		switch v := k.(type) {
		case tcell.Key:
			key = v
		case rune:
			ch = v
		}
		pending := pendingInput(s)
		r := core.HandleUnifiedKeyResult(s, key, ch, false, cfg, searchCols)
		if pick, ok := inputPick(pending, inputItem, r.Action); ok {
			return pick
		}
		action := processAction(s, r.Action)
		if pending == "" && pendingInput(s) != "" {
			inputItem = r.Item
		}
		if pick, ok := endingPick(action, r); ok {
			return pick
		}
	}
	return Pick{}
}

func newInputState(t *testing.T) (*core.State, Config, []int) {
	t.Helper()
	items := core.ParseLines([]string{"rename-window\tRename a window.\tnew-name", "kill-pane\tDestroy the given pane."}, "\t", false, false)
	for i := range items {
		if len(items[i].Fields) > 2 {
			items[i].Action = &core.ItemAction{Type: "input", Target: items[i].Fields[2]}
		}
	}
	cfg := Config{Layout: "reverse", HidePalette: true, TreeMode: true, Tiered: true, DepthPenalty: 5, AcceptNth: []int{1}}
	s, searchCols := core.NewState(items, cfg)
	ctx := s.TopCtx()
	ctx.TreeExpanded = make(map[int]bool)
	ctx.QueryExpanded = make(map[int]bool)
	ctx.TreeCursor = -1
	return s, cfg, searchCols
}

func TestInputLeafAsksForTextAndPrintsIt(t *testing.T) {
	s, cfg, searchCols := newInputState(t)
	if pick := pressKeys(t, s, cfg, searchCols, 'r', 'e', 'n', tcell.KeyEnter); pick.Kind != "" {
		t.Fatalf("Enter on an input leaf ended the picker with %+v", pick)
	}
	if s.PromptMode != inputMode || s.PromptPlaceholder != "new-name" {
		t.Fatalf("prompt mode %q, hint %q; want %q, %q", s.PromptMode, s.PromptPlaceholder, inputMode, "new-name")
	}
	s, cfg, searchCols = newInputState(t)
	pick := pressKeys(t, s, cfg, searchCols, 'r', 'e', 'n', tcell.KeyEnter, 'w', ':', 'x', tcell.KeyEnter)
	if pick.Kind != "input" || pick.Output != "rename-window\tw:x" || pick.Text != "w:x" || len(pick.Item.Fields) == 0 || pick.Item.Fields[0] != "rename-window" {
		t.Fatalf("pick = %+v, want the rename-window input leaf with text w:x", pick)
	}
}

func TestEscFromInputGoesBackToTheList(t *testing.T) {
	s, cfg, searchCols := newInputState(t)
	pressKeys(t, s, cfg, searchCols, 'r', 'e', 'n', tcell.KeyEnter, tcell.KeyEscape)
	if s.PromptMode != "" {
		t.Fatalf("Esc left prompt mode %q", s.PromptMode)
	}
	if got := string(s.TopCtx().Query); got != "ren" {
		t.Fatalf("query after Esc = %q, want the search kept", got)
	}
}

func TestPlainLeafStillPicksOnEnter(t *testing.T) {
	s, cfg, searchCols := newInputState(t)
	if pick := pressKeys(t, s, cfg, searchCols, 'k', 'i', 'l', 'l', tcell.KeyEnter); pick.Kind != "select" || pick.Output != "kill-pane" {
		t.Fatalf("pick = %+v, want kill-pane", pick)
	}
}

func TestPickIsTheItemNotItsName(t *testing.T) {
	items := []core.Item{
		{Fields: []string{"Steam"}, Action: &core.ItemAction{Type: "command", Target: "launch steam"}, ParentIdx: -1, PropertyOf: -1},
		{Fields: []string{"Steam"}, Action: &core.ItemAction{Type: "command", Target: "focus steam"}, ParentIdx: -1, PropertyOf: -1},
	}
	cfg := Config{Layout: "reverse", HidePalette: true, TreeMode: true, Tiered: true, DepthPenalty: 5, AcceptNth: []int{1}}
	s, searchCols := core.NewState(items, cfg)
	ctx := s.TopCtx()
	ctx.TreeExpanded = make(map[int]bool)
	ctx.QueryExpanded = make(map[int]bool)
	ctx.TreeCursor = -1
	pick := pressKeys(t, s, cfg, searchCols, tcell.KeyDown, tcell.KeyDown, tcell.KeyEnter)
	if pick.Kind != "select" || pick.Output != "Steam" || pick.Item.Action == nil || pick.Item.Action.Target != "focus steam" {
		t.Fatalf("pick = %+v, want the second Steam (focus steam)", pick)
	}
}
