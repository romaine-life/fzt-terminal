package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/romaine-life/fzt/core"
	frontend "github.com/romaine-life/fzt-frontend"
)

const syncInterval = 20 * 60 // 20 minutes in seconds

// syncIcon marks a newer menu on the server; syncFailedIcon, drawn red, marks a check
// that couldn't sign in or reach the menu.
const (
	syncIcon       = "⟳"
	syncFailedIcon = "!"
)

// checkMenuStale is frontend.CheckMenuStale; a seam for tests.
var checkMenuStale = frontend.CheckMenuStale

// initSyncCheck reads .last-sync-check, sets SyncNextCheck on state,
// and spawns a background goroutine if a check is due.
//
// The actual "fetch from API and compare timestamps" logic lives in
// frontend.CheckMenuStale. This file owns only the renderer-side
// concerns: the timer, UI state mutation (SyncIcon, title), and posting
// events back into the tcell event loop.
//
// A failed check (signed out, auth.romaine.life or the menu API unreachable)
// shows its reason in the title and the red icon, and isn't recorded, so the
// next menu open checks again rather than waiting out the interval.
func initSyncCheck(s *core.State, cfg Config, postEvent func()) {
	if cfg.ConfigDir == "" {
		return
	}

	lastCheckFile := filepath.Join(cfg.ConfigDir, ".last-sync-check")
	lastCheck := int64(0)
	if data, err := os.ReadFile(lastCheckFile); err == nil {
		lastCheck, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}

	now := time.Now().Unix()
	s.SyncNextCheck = lastCheck + syncInterval

	if now >= s.SyncNextCheck {
		go func() {
			stale, err := checkMenuStale(cfg.ConfigDir)
			if err != nil {
				s.SyncIcon = syncFailedIcon
				s.SetTitle(err.Error()+cacheAge(cfg.ConfigDir, time.Now()), 2)
				postEvent()
				return
			}
			if stale {
				s.SyncIcon = syncIcon
			}
			nowStr := strconv.FormatInt(time.Now().Unix(), 10)
			os.WriteFile(lastCheckFile, []byte(nowStr), 0644)
			s.SyncNextCheck = time.Now().Unix() + syncInterval
			postEvent()
		}()
	}
}

// cacheAge says how old the menu on screen is, for a failed check's title:
// " · menu last synced 7h ago".
func cacheAge(configDir string, now time.Time) string {
	info, err := os.Stat(filepath.Join(configDir, "menu-cache.yaml"))
	if err != nil {
		return " · no menu synced yet"
	}
	return " · menu last synced " + ago(now.Sub(info.ModTime())) + " ago"
}

func ago(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}
