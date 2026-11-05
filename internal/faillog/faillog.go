// Package faillog records fzt's failures in the file log-ship sends to the app log pile
// (at-home-tools docs/app-logs.md). Every home PC can read the pile, so no caller hands
// this a query, a pick, an item or a path. A line is one write, which keeps it whole
// when several fzts fail at once.
package faillog

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/romaine-life/fzt/render"
)

const limit = 1 << 20

// Write appends one line. A log that can't be written gives way to the failure it was
// reporting, which the caller has already shown.
func Write(what string) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	file := filepath.Join(dir, "fzt", "fzt.log")
	if info, err := os.Stat(file); err == nil && info.Size() > limit {
		_ = os.Rename(file, file+".old")
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line(what))
}

func line(what string) string {
	program, _ := os.Executable()
	what = strings.NewReplacer("\r", " ", "\n", " ").Replace(what)
	return fmt.Sprintf("%s %s %s pid %d: %s\n", time.Now().Format(time.RFC3339), filepath.Base(program), render.Version, os.Getpid(), what)
}

// Crash, deferred, logs a panic and lets it carry on, so the process dies as it always
// did. A runtime error's text is only numbers and type names, but another panic's value
// can hold anything, so that one is logged by its type. Frames carry no arguments.
func Crash() {
	r := recover()
	if r == nil {
		return
	}
	what := fmt.Sprintf("%T", r)
	if err, ok := r.(runtime.Error); ok {
		what = err.Error()
	}
	Write("crash: " + what + " at " + stack())
	panic(r)
}

func stack() string {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(3, pcs)])
	var names []string
	for {
		frame, more := frames.Next()
		if name := path.Base(frame.Function); !strings.HasPrefix(name, "runtime.") {
			names = append(names, fmt.Sprintf("%s:%d", name, frame.Line))
		}
		if !more {
			return strings.Join(names, " < ")
		}
	}
}
