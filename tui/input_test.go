package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
)

// pressKeys runs keys through the same steps as runWithSession and returns
// the pick that ended the picker, or "" if none did.
func pressKeys(t *testing.T, s *core.State, cfg Config, searchCols []int, keys ...any) string {
	t.Helper()
	for _, k := range keys {
		key, ch := tcell.KeyRune, rune(0)
		switch v := k.(type) {
		case tcell.Key:
			key = v
		case rune:
			ch = v
		}
		pending := pendingInput(s)
		raw := core.HandleUnifiedKey(s, key, ch, false, cfg, searchCols)
		if pick, ok := inputResult(pending, raw); ok {
			return pick
		}
		if action := processAction(s, raw); len(action) > 7 && action[:7] == "select:" {
			return action[7:]
		}
	}
	return ""
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
	if pick := pressKeys(t, s, cfg, searchCols, 'r', 'e', 'n', tcell.KeyEnter); pick != "" {
		t.Fatalf("Enter on an input leaf ended the picker with %q", pick)
	}
	if s.PromptMode != inputMode || s.PromptPlaceholder != "new-name" {
		t.Fatalf("prompt mode %q, hint %q; want %q, %q", s.PromptMode, s.PromptPlaceholder, inputMode, "new-name")
	}
	if pick := pressKeys(t, s, cfg, searchCols, 'w', ':', 'x', tcell.KeyEnter); pick != "rename-window\tw:x" {
		t.Fatalf("pick = %q, want %q", pick, "rename-window\tw:x")
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
	if pick := pressKeys(t, s, cfg, searchCols, 'k', 'i', 'l', 'l', tcell.KeyEnter); pick != "kill-pane" {
		t.Fatalf("pick = %q, want kill-pane", pick)
	}
}
