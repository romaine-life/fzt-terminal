package tui

import (
	"strings"
	"testing"
)

// promptLine is the frame's search box line.
func promptLine(t *testing.T, frame string) string {
	t.Helper()
	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "┌") && i+1 < len(lines) {
			return lines[i+1]
		}
	}
	t.Fatalf("no search box in frame:\n%s", frame)
	return ""
}

// StartFolder opens the menu inside a folder, its name the search box's
// breadcrumb, and Escape backs out to the root (SHELL-247: Ctrl+B opens the
// at menu inside its tmux folder).
func TestStartFolderOpensInside(t *testing.T) {
	cfg := Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, StartFolder: []string{"move-tab"}}

	frames := Simulate(revealItems(8, 6), cfg, "{esc}", 60, 20, false)
	if p := promptLine(t, frames[0].Content); !strings.Contains(p, "move-tab") {
		t.Fatalf("the menu didn't open inside move-tab; search box: %q", p)
	}
	if rows := strings.Join(listRows(t, frames[0].Content), "\n"); !strings.Contains(rows, "session-a") {
		t.Fatalf("move-tab isn't open in the tree:\n%s", rows)
	}
	if p := promptLine(t, frames[len(frames)-1].Content); strings.Contains(p, "move-tab") {
		t.Fatalf("Escape didn't back out to the root; search box: %q", p)
	}
}

func TestStartFolderMissingSaysSo(t *testing.T) {
	cfg := Config{Layout: "reverse", Border: true, Title: "at", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, StartFolder: []string{"tmux"}}
	frames := Simulate(revealItems(8, 6), cfg, "", 60, 20, false)
	if !strings.Contains(frames[0].Content, "no folder tmux in this menu") {
		t.Fatalf("a missing start folder isn't reported:\n%s", frames[0].Content)
	}
	if strings.Contains(promptLine(t, frames[0].Content), "tmux") {
		t.Fatalf("a missing start folder didn't leave the menu at the root:\n%s", frames[0].Content)
	}
}
