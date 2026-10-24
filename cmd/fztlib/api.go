package main

import (
	"encoding/json"
	"sync"

	"github.com/romaine-life/fzt-terminal/host"
)

// response is every call's reply: the session id from open, the frame to draw, or the error.
type response struct {
	Session int         `json:"session,omitempty"`
	Frame   *host.Frame `json:"frame,omitempty"`
	Error   string      `json:"error,omitempty"`
}

var (
	mu       sync.Mutex
	sessions = map[int]*host.Session{}
	nextID   = 1
)

func reply(r response) string {
	data, err := json.Marshal(r)
	if err != nil {
		data, _ = json.Marshal(response{Error: err.Error()})
	}
	return string(data)
}

func fail(err error) string {
	return reply(response{Error: err.Error()})
}

func open(menu, options string) string {
	var opts host.Options
	if err := json.Unmarshal([]byte(options), &opts); err != nil {
		return fail(err)
	}
	s, err := host.Open(menu, opts)
	if err != nil {
		return fail(err)
	}
	mu.Lock()
	id := nextID
	nextID++
	sessions[id] = s
	mu.Unlock()
	f := s.Frame()
	return reply(response{Session: id, Frame: &f})
}

// with runs fn on an open session, one call at a time
func with(id int, fn func(*host.Session) (host.Frame, error)) string {
	mu.Lock()
	defer mu.Unlock()
	s, ok := sessions[id]
	if !ok {
		return reply(response{Error: "no such session"})
	}
	f, err := fn(s)
	if err != nil {
		return fail(err)
	}
	return reply(response{Frame: &f})
}

func key(id int, event string) string {
	return with(id, func(s *host.Session) (host.Frame, error) {
		var ev host.KeyEvent
		if err := json.Unmarshal([]byte(event), &ev); err != nil {
			return host.Frame{}, err
		}
		return s.Key(ev)
	})
}

func resize(id, width, height int) string {
	return with(id, func(s *host.Session) (host.Frame, error) {
		return s.Resize(width, height), nil
	})
}

func prompt(id int, p string) string {
	return with(id, func(s *host.Session) (host.Frame, error) {
		var pr host.Prompt
		if err := json.Unmarshal([]byte(p), &pr); err != nil {
			return host.Frame{}, err
		}
		return s.Prompt(pr), nil
	})
}

func closeSession(id int) {
	mu.Lock()
	delete(sessions, id)
	mu.Unlock()
}
