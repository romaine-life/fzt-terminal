package main

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt-terminal/tui"
	"github.com/romaine-life/fzt/core"
)

func TestPickerRendersStdinListInTreeStyle(t *testing.T) {
	items := core.ParseLines([]string{"journal-biking", "journal-career", "journal-dreams", "lights"}, "\t", false, false)
	cfg := tui.Config{Layout: "reverse", Border: true, HidePalette: true, TreeMode: true, Tiered: true, DepthPenalty: 5}
	frame := tui.FormatFrames(tui.Simulate(items, cfg, "", 50, 10, false))
	t.Log("\n" + frame)
	for _, want := range []string{"type to search", "journal-dreams", "lights"} {
		if !strings.Contains(frame, want) {
			t.Fatalf("frame missing %q", want)
		}
	}
}

func TestDirectoryLinesGetFolderIcon(t *testing.T) {
	dir := t.TempDir()
	if !isDir(dir) || !isDir("'"+dir+"'") {
		t.Fatalf("isDir(%q) = false", dir)
	}
	if isDir(dir + `\missing`) {
		t.Fatal("isDir reported a missing path as a directory")
	}

	items := core.ParseLines([]string{dir, "not-a-dir"}, "\t", false, false)
	items[0].IsDir = isDir(items[0].Fields[0])
	cfg := tui.Config{Layout: "reverse", Border: true, HidePalette: true, TreeMode: true, Tiered: true, DepthPenalty: 5}
	frame := tui.FormatFrames(tui.Simulate(items, cfg, "", 200, 8, false))
	if strings.Count(frame, "\U000F024B") != 1 || strings.Count(frame, "") != 1 {
		t.Fatalf("want one folder icon and one file icon:\n%s", frame)
	}
}

func TestTieredRowsOpenAsSubMenus(t *testing.T) {
	// The tmux palette's shape: a header, then a folder whose first child follows it.
	lines := []string{"0\tmove-tab\tMove the tab", "1\twork\tsession\tid-work", "1\tplay\tsession\tid-play", "0\tkill-pane\tDestroy the pane\tid-kill"}
	cfg := tui.Config{Layout: "reverse", HidePalette: true, Tiered: true, DepthPenalty: 5, Delimiter: "\t", AcceptNth: []int{3}}
	items := stdinItems(lines, &cfg, "name\tdescription", 0, 0)
	data := items[cfg.HeaderLines:]
	if !data[0].HasChildren || data[1].ParentIdx != 0 || data[2].ParentIdx != 0 || data[3].ParentIdx != -1 {
		t.Fatalf("tree not linked: %+v", data)
	}
	// A parent loop made the first search spin forever (FZT-52): search must return.
	done := make(chan struct{})
	go func() { tui.Simulate(items, cfg, "w", 80, 20, false); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("searching hung")
	}
	s, searchCols := core.NewState(items, cfg)
	ctx := s.TopCtx()
	ctx.TreeExpanded = make(map[int]bool)
	ctx.QueryExpanded = make(map[int]bool)
	ctx.TreeCursor = -1
	press := func(keys ...any) string {
		for _, k := range keys {
			key, ch := tcell.KeyRune, rune(0)
			switch v := k.(type) {
			case tcell.Key:
				key = v
			case rune:
				ch = v
			}
			raw := core.HandleUnifiedKey(s, key, ch, false, cfg, searchCols)
			if a := tui.ProcessAction(s, raw); strings.HasPrefix(a, "select:") {
				return strings.TrimPrefix(a, "select:")
			}
		}
		return ""
	}
	if pick := press('m', 'o', 'v', 'e', tcell.KeyEnter); pick != "" {
		t.Fatalf("Enter on a folder picked %q", pick)
	}
	if len(s.TopCtx().Scope) != 2 {
		t.Fatalf("Enter on a folder didn't open it: scope %d", len(s.TopCtx().Scope))
	}
	press(tcell.KeyEscape)
	if len(s.TopCtx().Scope) != 1 {
		t.Fatalf("Esc didn't go back: scope %d", len(s.TopCtx().Scope))
	}
	press(tcell.KeyEnter)
	if pick := press('p', 'l', 'a', 'y', tcell.KeyEnter); pick != "id-play" {
		t.Fatalf("pick in the sub-menu = %q, want id-play", pick)
	}
}

func TestTagSearchShowsTagsInDescriptionColumn(t *testing.T) {
	// tmux palette rows: depth, name, description, hint, id, tags.
	lines := []string{
		"0\tmove-window\tMove a window\t\tid-1\tmove window",
		"0\tswitch-client\tSwitch the session\t\tid-2\tsession,client,switch",
		"0\tkill-session\tDestroy a session\t\tid-3\tsession kill",
	}
	cfg := tui.Config{Layout: "reverse", HidePalette: true, Tiered: true, DepthPenalty: 5, Delimiter: "\t", TagMode: true}
	items := stdinItems(lines, &cfg, "name\tdescription", 0, 5)
	if got := items[cfg.HeaderLines+1].Tags; strings.Join(got, "|") != "session|client|switch" {
		t.Fatalf("tags = %q", got)
	}
	frames := tui.Simulate(items, cfg, "sess", 100, 10, false)
	last := frames[len(frames)-1].Content
	for _, want := range []string{"switch-client", "kill-session", "session client switch", "ion", "tags"} {
		if !strings.Contains(last, want) {
			t.Fatalf("frame lacks %q:\n%s", want, last)
		}
	}
	if strings.Contains(last, "move-window") || strings.Contains(last, "Switch the session") {
		t.Fatalf("frame shows a non-match or a description:\n%s", last)
	}
	frames = tui.Simulate(items, cfg, "sess zzz", 100, 10, false)
	if last := frames[len(frames)-1].Content; !strings.Contains(last, "no tag: zzz") {
		t.Fatalf("unresolved term not reported:\n%s", last)
	}
}
