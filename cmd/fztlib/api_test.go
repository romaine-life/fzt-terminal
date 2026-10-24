package main

import (
	"encoding/json"
	"testing"
)

func decode(t *testing.T, s string) response {
	t.Helper()
	var r response
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		t.Fatalf("%v: %s", err, s)
	}
	if r.Error != "" {
		t.Fatalf("error: %s", r.Error)
	}
	return r
}

func TestOpenKeyAndClose(t *testing.T) {
	r := decode(t, open(`- {name: "next", description: "Next track", action: "NextTrack"}`, `{"title":"spt","width":60,"height":10}`))
	if r.Session == 0 || r.Frame == nil || len(r.Frame.Rows) == 0 {
		t.Fatalf("open = %+v", r)
	}
	for _, c := range "next" {
		decode(t, key(r.Session, `{"key":"char","char":"`+string(c)+`"}`))
	}
	got := decode(t, key(r.Session, `{"key":"enter"}`))
	if got.Frame.Result == nil || got.Frame.Result.Action != "NextTrack" {
		t.Fatalf("result = %+v", got.Frame.Result)
	}
	closeSession(r.Session)
	var after response
	json.Unmarshal([]byte(key(r.Session, `{"key":"enter"}`)), &after)
	if after.Error == "" {
		t.Fatal("closed session still answers")
	}
}

func TestBadInputIsAnError(t *testing.T) {
	var r response
	json.Unmarshal([]byte(open("- {name: x}", "not json")), &r)
	if r.Error == "" {
		t.Fatal("no error for bad options")
	}
}
