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
