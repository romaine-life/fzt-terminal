package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/romaine-life/fzt/core"
)

func runSyncCheck(t *testing.T, stale bool, err error) (*core.State, string) {
	t.Helper()
	old := checkMenuStale
	checkMenuStale = func(string) (bool, error) { return stale, err }
	t.Cleanup(func() { checkMenuStale = old })

	dir := t.TempDir()
	s := &core.State{}
	done := make(chan struct{})
	initSyncCheck(s, Config{ConfigDir: dir}, func() { close(done) })
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("background check never finished")
	}
	return s, filepath.Join(dir, ".last-sync-check")
}

func TestSyncCheckFailureIsShown(t *testing.T) {
	s, lastCheck := runSyncCheck(t, false, errors.New("signed out: this PC's registration was revoked at auth.romaine.life/admin"))
	if s.SyncIcon != syncFailedIcon {
		t.Errorf("icon = %q, want the failure icon", s.SyncIcon)
	}
	if s.TitleStyle != 2 || s.TitleOverride == "" {
		t.Errorf("title = %q (style %d), want the reason as an error", s.TitleOverride, s.TitleStyle)
	}
	if want := "· no menu synced yet"; !strings.HasSuffix(s.TitleOverride, want) {
		t.Errorf("title %q doesn't say how old the menu is", s.TitleOverride)
	}
	if _, err := os.Stat(lastCheck); err == nil {
		t.Error("a failed check was recorded, so the next open would wait out the interval")
	}
}

func TestSyncCheckStaleShowsIcon(t *testing.T) {
	s, lastCheck := runSyncCheck(t, true, nil)
	if s.SyncIcon != syncIcon || s.TitleOverride != "" {
		t.Errorf("icon = %q title = %q", s.SyncIcon, s.TitleOverride)
	}
	if _, err := os.Stat(lastCheck); err != nil {
		t.Error("a good check wasn't recorded")
	}
}

func TestSyncCheckCurrentIsQuiet(t *testing.T) {
	s, _ := runSyncCheck(t, false, nil)
	if s.SyncIcon != "" || s.TitleOverride != "" {
		t.Errorf("icon = %q title = %q, want nothing shown", s.SyncIcon, s.TitleOverride)
	}
}

func TestCacheAge(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "menu-cache.yaml")
	os.WriteFile(cache, nil, 0o644)
	now := time.Now()
	for age, want := range map[time.Duration]string{
		12 * time.Minute: " · menu last synced 12m ago",
		7 * time.Hour:    " · menu last synced 7h ago",
		75 * time.Hour:   " · menu last synced 3 days ago",
	} {
		os.Chtimes(cache, now.Add(-age), now.Add(-age))
		if got := cacheAge(dir, now); got != want {
			t.Errorf("age %v: %q, want %q", age, got, want)
		}
	}
}
