// Package host runs a headless fzt tree session for an app that draws the menu itself: the app
// sends key events and gets back what to draw as data, and what the user picked. cmd/fztlib
// exposes it as a C library so apps in other languages can load fzt in-process.
package host

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/romaine-life/fzt-frontend"
	"github.com/romaine-life/fzt-terminal/tui"
	"github.com/romaine-life/fzt/core"
	"github.com/romaine-life/fzt/render"
)

// Options configure a session.
type Options struct {
	Title  string `json:"title"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	// Name shown for the host in the `:` palette
	Frontend string `json:"frontend"`
	// The host's own commands, listed first in the `:` palette
	Commands []Command `json:"commands"`
}

// Command is a host command in the `:` palette. Picking it returns a "command" result with its
// action.
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// KeyEvent is one key press. Key names a special key ("enter", "esc", "up", "down", "left",
// "right", "tab", "backtab", "backspace", "delete", "home", "end", "pgup", "pgdn"), or is "char"
// with the typed character in Char.
type KeyEvent struct {
	Key   string `json:"key"`
	Char  string `json:"char"`
	Shift bool   `json:"shift"`
}

// Prompt switches the text box to take free text for a picked entry, like a search term.
// Enter returns a "prompt" result with Action and the typed text.
type Prompt struct {
	Mode        string `json:"mode"`
	Action      string `json:"action"`
	Icon        string `json:"icon"`
	Placeholder string `json:"placeholder"`
}

// Frame is what to draw, and the user's pick once there is one.
type Frame struct {
	Rows   []render.VisibleRow `json:"rows"`
	Prompt render.PromptState  `json:"prompt"`
	UI     render.UIState      `json:"ui"`
	Result *Result             `json:"result,omitempty"`
}

// Result is what the user picked. Kind is "select" (a menu entry; Action is its action),
// "prompt" (Action is the prompt's action, Text what was typed), "command" (a host command from
// the `:` palette) or "cancel".
type Result struct {
	Kind   string `json:"kind"`
	Action string `json:"action,omitempty"`
	Text   string `json:"text,omitempty"`
	Name   string `json:"name,omitempty"`
}

// Session is one open menu.
type Session struct {
	sess *render.Session
}

// Open starts a session over a menu in fzt's YAML tree format.
func Open(menu string, opts Options) (*Session, error) {
	items, err := core.LoadYAMLFromString(menu)
	if err != nil {
		return nil, fmt.Errorf("loading menu: %w", err)
	}
	commands := make([]core.CommandItem, len(opts.Commands))
	for i, c := range opts.Commands {
		commands[i] = core.CommandItem{Name: c.Name, Description: c.Description, Action: c.Action}
	}
	cfg := tui.Config{
		TreeMode:         true,
		Tiered:           true,
		DepthPenalty:     5,
		Title:            opts.Title,
		FrontendName:     opts.Frontend,
		FrontendCommands: commands,
		EnvTags:          []string{"host"},
	}
	return &Session{sess: tui.NewTreeSession(items, cfg, opts.Width, opts.Height)}, nil
}

// Frame returns what to draw now.
func (s *Session) Frame() Frame {
	s.sess.Render()
	return s.frame(nil)
}

// Resize changes the size the menu is laid out for.
func (s *Session) Resize(width, height int) Frame {
	s.sess.Resize(width, height)
	return s.frame(nil)
}

// Key handles a key press.
func (s *Session) Key(ev KeyEvent) (Frame, error) {
	key, ch, err := tcellKey(ev)
	if err != nil {
		return Frame{}, err
	}
	_, action := s.sess.HandleKeyResult(key, ch, ev.Shift)
	return s.frame(s.result(action)), nil
}

// Prompt switches the text box to free text, for the entry just picked.
func (s *Session) Prompt(p Prompt) Frame {
	var icon rune
	if r := []rune(p.Icon); len(r) > 0 {
		icon = r[0]
	}
	s.sess.State().EnterPromptMode(p.Mode, p.Action, icon, p.Placeholder)
	s.sess.Render()
	return s.frame(nil)
}

func (s *Session) frame(result *Result) Frame {
	return Frame{
		Rows:   s.sess.GetVisibleRows(),
		Prompt: s.sess.GetPromptState(),
		UI:     s.sess.GetUIState(),
		Result: result,
	}
}

func (s *Session) result(r render.SessionAction) *Result {
	switch {
	case r.Action == "":
		return nil
	case r.Action == "cancel":
		return &Result{Kind: "cancel"}
	case strings.HasPrefix(r.Action, "prompt:"):
		action, text, _ := strings.Cut(strings.TrimPrefix(r.Action, "prompt:"), ":")
		return &Result{Kind: "prompt", Action: action, Text: text}
	case strings.HasPrefix(r.Action, "select:") && r.HasItem:
		name := ""
		if len(r.Item.Fields) > 0 {
			name = r.Item.Fields[0]
		}
		// the `:` palette's own commands run here, as in the full-screen picker
		if r.Item.Injected {
			action := frontend.HandleCommandAction(s.sess.State(), r.Item)
			if action == "" {
				return nil
			}
			return &Result{Kind: "command", Action: action, Name: name}
		}
		result := &Result{Kind: "select", Name: name}
		if r.Item.Action != nil {
			result.Action = r.Item.Action.Target
		}
		return result
	default:
		return &Result{Kind: "command", Action: r.Action}
	}
}

var namedKeys = map[string]tcell.Key{
	"enter":     tcell.KeyEnter,
	"esc":       tcell.KeyEscape,
	"up":        tcell.KeyUp,
	"down":      tcell.KeyDown,
	"left":      tcell.KeyLeft,
	"right":     tcell.KeyRight,
	"tab":       tcell.KeyTab,
	"backtab":   tcell.KeyBacktab,
	"backspace": tcell.KeyBackspace2,
	"delete":    tcell.KeyDelete,
	"home":      tcell.KeyHome,
	"end":       tcell.KeyEnd,
	"pgup":      tcell.KeyPgUp,
	"pgdn":      tcell.KeyPgDn,
}

func tcellKey(ev KeyEvent) (tcell.Key, rune, error) {
	if ev.Key == "char" {
		r := []rune(ev.Char)
		if len(r) != 1 {
			return 0, 0, fmt.Errorf("char key needs one character, got %q", ev.Char)
		}
		return tcell.KeyRune, r[0], nil
	}
	if key, ok := namedKeys[ev.Key]; ok {
		return key, 0, nil
	}
	return 0, 0, fmt.Errorf("unknown key %q", ev.Key)
}
