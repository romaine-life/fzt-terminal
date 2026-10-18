package main

import (
	"strings"
	"testing"

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
