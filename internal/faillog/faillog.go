// Package faillog records fzt's failures in the file log-ship sends to the app log pile
// (at-home-tools docs/app-logs.md). Every home PC can read the pile, so no caller hands
// this a query, a pick, an item or a path. A line is one write, which keeps it whole
// when several fzts fail at once, and the file rolls to fzt.log.old past 1 MiB.
package faillog

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/romaine-life/fzt/render"
	"github.com/romaine-life/logline"
)

// Env names a file to log to instead of %LOCALAPPDATA%\fzt\fzt.log: tests and
// throwaway runs keep their lines out of the pile with it.
const Env = "FZT_LOG"

// Write logs one failure as an error line (logline's standard, AHT-204): msg is fixed
// wording, kv its key, value fields, then the program and fzt's version. The log is
// opened for this line only, so a host that loaded fztlib holds no state of ours
// between calls, and a log that can't be written gives way to the failure it was
// reporting, which the caller has already shown.
func Write(msg string, kv ...any) {
	program, _ := os.Executable()
	kv = append(kv, "exe", filepath.Base(program), "version", render.Version)
	logline.Open("fzt", Env).Error(msg, kv...)
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
	Write("crash", "error", what, "stack", stack())
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
