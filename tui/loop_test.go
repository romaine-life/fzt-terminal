package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt/core"
)

const loopYAML = `
- name: lightson
  action: lightson
  quick: true
- name: ollamastop
  action: ollamastop
  quick: true
  input: model to stop
`

// loopHarness runs sessionLoop on a simulated screen in the background.
type loopHarness struct {
	t      *testing.T
	screen tcell.SimulationScreen
	live   *Live
	picks  chan Pick
	done   chan struct{}
}

func startLoop(t *testing.T, yaml string, cfg Config) *loopHarness {
	t.Helper()
	items, err := core.LoadYAMLFromString(yaml)
	if err != nil {
		t.Fatal(err)
	}
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	screen.SetSize(60, 20)
	h := &loopHarness{t: t, screen: screen, live: &Live{screen: screen}, picks: make(chan Pick, 8), done: make(chan struct{})}
	go func() {
		defer close(h.done)
		sessionLoop(screen, items, cfg, h.live, func(p Pick) bool {
			h.picks <- p
			return p.Kind == "command" && p.Output == "stop-test"
		})
	}()
	t.Cleanup(func() {
		select {
		case <-h.done:
		default:
			screen.Fini()
		}
	})
	return h
}

func (h *loopHarness) keys(text string) {
	for _, r := range text {
		h.screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
	}
}

func (h *loopHarness) key(k tcell.Key) { h.screen.InjectKey(k, 0, tcell.ModNone) }

func (h *loopHarness) pick() Pick {
	h.t.Helper()
	select {
	case p := <-h.picks:
		return p
	case <-time.After(2 * time.Second):
		h.t.Fatal("no pick")
	}
	return Pick{}
}

func (h *loopHarness) noPick() {
	h.t.Helper()
	select {
	case p := <-h.picks:
		h.t.Fatalf("unexpected pick %+v", p)
	case <-time.After(300 * time.Millisecond):
	}
}

// row returns the text of screen row y once the loop has drawn it.
func (h *loopHarness) waitRow(y int, want string) {
	h.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		cells, w, _ := h.screen.GetContents()
		var b strings.Builder
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) > 0 {
				b.WriteRune(c.Runes[0])
			}
		}
		if got = b.String(); strings.Contains(got, want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	h.t.Fatalf("row %d = %q, want it to contain %q", y, got, want)
}

func loopCfg() Config {
	return Config{Layout: "reverse", Tiered: true, TreeMode: true, DepthPenalty: 5, HidePalette: true, ShowPanel: true}
}

func TestLoopKeepsGoingAfterPicksAndCancel(t *testing.T) {
	h := startLoop(t, loopYAML, loopCfg())
	h.keys("lightson")
	h.key(tcell.KeyEnter)
	if p := h.pick(); p.Kind != "select" || p.Item.Fields[0] != "lightson" || !p.Item.Quick {
		t.Fatalf("first pick = %+v", p)
	}
	// Escape steps out of the search, then clears it; at the top, where a
	// one-shot Run would end, the loop hands the cancel to the frontend and
	// carries on until the frontend closes it.
	h.keys("light")
	h.key(tcell.KeyEscape)
	h.key(tcell.KeyEscape)
	h.noPick()
	h.key(tcell.KeyEscape)
	if p := h.pick(); p.Kind != "cancel" {
		t.Fatalf("Escape at the top = %+v, want a cancel", p)
	}
	h.keys("lightson")
	h.key(tcell.KeyEnter)
	if p := h.pick(); p.Kind != "select" || p.Item.Fields[0] != "lightson" {
		t.Fatalf("pick after cancel = %+v", p)
	}
}

func TestLoopPanelShowsLiveResults(t *testing.T) {
	h := startLoop(t, loopYAML, loopCfg())
	// No border: the panel box is rows 0-4 (its lines are 1-3), the prompt
	// bar starts at row 5.
	h.live.Update(func(s *core.State) { s.SetPanel("Living room lights on\nsecond\nthird\nfourth", core.PanelOK) })
	h.waitRow(0, "┌")
	h.waitRow(1, "│ Living room lights on")
	h.waitRow(3, "third …")
	h.waitRow(4, "└")
	h.waitRow(5, "┌")
	h.live.Update(func(s *core.State) { s.SetPanel("done", core.PanelFailed) })
	h.waitRow(1, "done")
	cells, w, _ := h.screen.GetContents()
	if r := cells[3*w+2].Runes; len(r) > 0 && r[0] != ' ' {
		t.Fatalf("a shorter result must clear the old lines, row 3 starts %q", string(r))
	}
}

func TestLoopFreezeDropsKeysAndEscapeAborts(t *testing.T) {
	h := startLoop(t, loopYAML, loopCfg())
	aborted := make(chan struct{}, 1)
	h.live.Update(func(s *core.State) { s.Freeze() })
	h.live.OnAbort(func() { aborted <- struct{}{} })
	h.keys("lightson")
	h.key(tcell.KeyEnter)
	h.noPick()
	h.key(tcell.KeyEscape)
	select {
	case <-aborted:
	case <-time.After(2 * time.Second):
		t.Fatal("Escape while frozen didn't call the abort")
	}
	h.live.OnAbort(nil)
	h.live.Update(func(s *core.State) { s.Unfreeze() })
	h.keys("lightson")
	h.key(tcell.KeyEnter)
	if p := h.pick(); p.Item.Fields[0] != "lightson" {
		t.Fatalf("pick after unfreeze = %+v", p)
	}
}

func TestDeclaredInputIsAskedBeforeThePick(t *testing.T) {
	h := startLoop(t, loopYAML, loopCfg())
	h.keys("ollamastop")
	h.key(tcell.KeyEnter)
	h.noPick()
	h.waitRow(core.PanelRows+1, "model to stop")
	h.keys("llama3.1")
	h.key(tcell.KeyEnter)
	p := h.pick()
	if p.Kind != "input" || p.Text != "llama3.1" || p.Item.Input != "model to stop" {
		t.Fatalf("pick = %+v", p)
	}
}

func TestReloadKeepsThePanel(t *testing.T) {
	h := startLoop(t, loopYAML, loopCfg())
	h.live.Update(func(s *core.State) { s.SetPanel("synced", core.PanelOK) })
	items, _ := core.LoadYAMLFromString("- name: lightsoff\n  action: lightsoff\n")
	h.live.Reload(items)
	h.waitRow(1, "synced")
	h.keys("lightsoff")
	h.key(tcell.KeyEnter)
	if p := h.pick(); p.Item.Fields[0] != "lightsoff" {
		t.Fatalf("pick after reload = %+v", p)
	}
}

func TestLoopCloseEndsTheMenu(t *testing.T) {
	h := startLoop(t, loopYAML, loopCfg())
	h.key(tcell.KeyEscape)
	if p := h.pick(); p.Kind != "cancel" {
		t.Fatalf("Escape at the top = %+v, want a cancel", p)
	}
	h.live.Close()
	select {
	case <-h.done:
	case <-time.After(2 * time.Second):
		t.Fatal("the menu kept running after Close")
	}
}
