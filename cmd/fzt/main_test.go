package main

import (
	"strings"
	"testing"

	"github.com/romaine-life/fzt-terminal/tui"
	"github.com/romaine-life/fzt/core"
)

func TestPickerRendersStdinList(t *testing.T) {
	items := core.ParseLines([]string{"journal-biking", "journal-career", "journal-dreams", "lights"}, "\t", false, false)
	cfg := tui.Config{Layout: "reverse", Border: true, Prompt: "> ", HidePalette: true, DepthPenalty: 5}
	frames := tui.Simulate(items, cfg, "dre", 50, 10, false)
	t.Log("\n" + tui.FormatFrames(frames[len(frames)-1:]))
	if !strings.Contains(tui.FormatFrames(frames[len(frames)-1:]), "journal-dreams") {
		t.Fatal("missing match")
	}
}
