package host

import "testing"

const menu = `
- name: "radio"
  description: "Radios"
  children:
    - name: "ambient electronic"
      description: "Radio from a random song in ambient electronic"
      action: "radio-random:ambient"
    - name: "from this song"
      description: "Radio from the song playing now"
      action: "radio-current"
- name: "search"
  description: "Search Spotify"
  action: "search"
`

func open(t *testing.T) *Session {
	t.Helper()
	s, err := Open(menu, Options{
		Title:    "spt",
		Width:    80,
		Height:   20,
		Frontend: "spt",
		Commands: []Command{{Name: "reload menu", Description: "Read the menu file again", Action: "reload-menu"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func typeText(t *testing.T, s *Session, text string) Frame {
	t.Helper()
	var f Frame
	for _, r := range text {
		var err error
		if f, err = s.Key(KeyEvent{Key: "char", Char: string(r)}); err != nil {
			t.Fatal(err)
		}
		if f.Result != nil {
			t.Fatalf("typing %q gave a result: %+v", text, f.Result)
		}
	}
	return f
}

func press(t *testing.T, s *Session, key string) Frame {
	t.Helper()
	f, err := s.Key(KeyEvent{Key: key})
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestOpenShowsTheTopLevel(t *testing.T) {
	f := open(t).Frame()
	var names []string
	for _, r := range f.Rows {
		names = append(names, r.Name)
	}
	if len(names) < 2 || names[0] != "radio" || names[1] != "search" {
		t.Fatalf("rows = %v", names)
	}
	if f.UI.Title != "spt" || f.Result != nil {
		t.Fatalf("ui = %+v, result = %+v", f.UI, f.Result)
	}
}

func TestSearchAndEnterSelectsTheEntrysAction(t *testing.T) {
	s := open(t)
	f := typeText(t, s, "amb")
	if f.Prompt.Query != "amb" {
		t.Fatalf("query = %q", f.Prompt.Query)
	}
	f = press(t, s, "enter")
	want := Result{Kind: "select", Action: "radio-random:ambient", Name: "ambient electronic"}
	if f.Result == nil || *f.Result != want {
		t.Fatalf("result = %+v, want %+v", f.Result, want)
	}
}

func TestPromptReturnsTheTypedText(t *testing.T) {
	s := open(t)
	typeText(t, s, "search")
	if f := press(t, s, "enter"); f.Result == nil || f.Result.Action != "search" {
		t.Fatalf("result = %+v", f.Result)
	}
	if f := s.Prompt(Prompt{Mode: "search", Action: "spotify-search", Placeholder: "search Spotify…"}); f.Prompt.Hint != "search Spotify…" {
		t.Fatalf("prompt = %+v", f.Prompt)
	}
	typeText(t, s, "boards of canada")
	want := Result{Kind: "prompt", Action: "spotify-search", Text: "boards of canada"}
	if f := press(t, s, "enter"); f.Result == nil || *f.Result != want {
		t.Fatalf("result = %+v, want %+v", f.Result, want)
	}
}

func TestHostCommandFromThePalette(t *testing.T) {
	s := open(t)
	typeText(t, s, "reload menu")
	want := Result{Kind: "command", Action: "reload-menu", Name: "reload menu"}
	if f := press(t, s, "enter"); f.Result == nil || *f.Result != want {
		t.Fatalf("result = %+v, want %+v", f.Result, want)
	}
}

func TestEscapeAtTheTopCancels(t *testing.T) {
	s := open(t)
	if f := press(t, s, "esc"); f.Result == nil || f.Result.Kind != "cancel" {
		t.Fatalf("result = %+v", f.Result)
	}
}

func TestUnknownKeyIsAnError(t *testing.T) {
	if _, err := open(t).Key(KeyEvent{Key: "f13"}); err == nil {
		t.Fatal("no error")
	}
}
