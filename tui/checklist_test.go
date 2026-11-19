package tui

import (
	"strings"
	"testing"

	"github.com/romaine-life/fzt/core"
)

// A checklist folder's rows draw a box in place of their icon, ticked or
// not, the search box counts the ticked rows, and the rows around the folder
// draw as usual.
func TestChecklistDrawsBoxesAndCount(t *testing.T) {
	items, err := core.LoadYAMLFromString(`
- name: senders
  checklist: true
  children:
    - name: news@a.com
    - name: bills@d.com
      ticked: true
- name: unsubscribe
  action: unsubscribe
`)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true}
	frames := Simulate(items, cfg, "news", 60, 10, false)
	frame := frames[len(frames)-1].Content
	for _, want := range []string{
		string(UntickedIcon) + "  news@a.com",
		string(TickedIcon) + "  bills@d.com",
		string(TickedIcon) + " 1 ticked",
		"  unsubscribe",
	} {
		if !strings.Contains(frame, want) {
			t.Fatalf("no %q in\n%s", want, frame)
		}
	}
}
