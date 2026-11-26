package faillog

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/romaine-life/fzt/render"
	"github.com/romaine-life/logline"
)

const (
	workerEnv   = "FZT_FAILLOG_TEST_WORKER"
	crashEnv    = "FZT_FAILLOG_TEST_CRASH"
	workerLines = 100
)

// TestMain makes the test binary a worker when a test starts it with one of these
// variables set, so a child never runs the tests.
func TestMain(m *testing.M) {
	if id := os.Getenv(workerEnv); id != "" {
		for i := range workerLines {
			Write("worker line", "worker", id, "line", i, "pad", strings.Repeat("x", 1000))
		}
		return
	}
	switch os.Getenv(crashEnv) {
	case "index":
		defer Crash()
		var none []int
		_ = none[len(os.Args)+3]
	case "value":
		defer Crash()
		panic(`C:\private\picked-item`)
	default:
		os.Exit(m.Run())
	}
}

// useCache points the log at a file of the test's own, through Env.
func useCache(t *testing.T) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "fzt.log")
	t.Setenv(Env, file)
	return file
}

// entry reads text, which must be one line, as logline's standard.
func entry(t *testing.T, text string) logline.Entry {
	t.Helper()
	if strings.Count(text, "\n") != 1 || !strings.HasSuffix(text, "\n") {
		t.Fatalf("not one line: %q", text)
	}
	e, err := logline.Parse(strings.TrimSuffix(text, "\n"))
	if err != nil {
		t.Fatalf("%v: %q", err, text)
	}
	return e
}

func field(e logline.Entry, key string) string {
	for _, f := range e.Fields {
		if f.Key == key {
			return f.Value
		}
	}
	return ""
}

func TestALineIsTheStandardsErrorLineWithProgramAndVersion(t *testing.T) {
	file := useCache(t)
	Write("couldn't read input", "error", "first\r\nsecond")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	e := entry(t, string(b))
	if e.Level != logline.Error || e.Message != "couldn't read input" || e.PID != os.Getpid() {
		t.Fatalf("%+v", e)
	}
	if field(e, "error") != "first\r\nsecond" || field(e, "version") != render.Version || field(e, "exe") == "" {
		t.Fatalf("%+v", e)
	}
}

// Without the override the log is %LOCALAPPDATA%\fzt\fzt.log, the file appLogs ships.
func TestTheLogStaysInTheCacheFolder(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(Env, "")
	t.Setenv("LocalAppData", dir)
	t.Setenv("XDG_CACHE_HOME", dir)
	Write("bad flag")
	b, err := os.ReadFile(filepath.Join(dir, "fzt", "fzt.log"))
	if err != nil {
		t.Fatal(err)
	}
	if e := entry(t, string(b)); e.Message != "bad flag" {
		t.Fatalf("%+v", e)
	}
}

func TestPastTheLimitTheLogRolls(t *testing.T) {
	file := useCache(t)
	if err := os.WriteFile(file, make([]byte, logline.Limit+1), 0o600); err != nil {
		t.Fatal(err)
	}
	Write("after the roll")
	if info, err := os.Stat(file + ".old"); err != nil || info.Size() != logline.Limit+1 {
		t.Fatalf("old file: %v %v", info, err)
	}
	if info, err := os.Stat(file); err != nil || info.Size() > 300 {
		t.Fatalf("new file: %v %v", info, err)
	}
}

func runWorkers(t *testing.T, count int) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for id := range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(exe)
			cmd.Env = append(os.Environ(), workerEnv+"="+strconv.Itoa(id+1))
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("worker %d: %v %s", id, err, out)
			}
		}()
	}
	wg.Wait()
}

// isWorkerLine says whether text is a worker's whole line, read as the standard.
func isWorkerLine(text string) bool {
	e, err := logline.Parse(text)
	return err == nil && e.Message == "worker line" && len(field(e, "pad")) == 1000
}

// wholeLines fails on any line a concurrent write broke, and counts the workers' lines.
func wholeLines(t *testing.T, file string) int {
	t.Helper()
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines := bufio.NewScanner(f)
	lines.Buffer(make([]byte, logline.Limit), logline.Limit)
	seen := 0
	for lines.Scan() {
		switch text := lines.Text(); {
		case text == "filler":
		case isWorkerLine(text):
			seen++
		default:
			t.Fatalf("broken line in %s: %.120q", file, text)
		}
	}
	if err := lines.Err(); err != nil {
		t.Fatal(err)
	}
	return seen
}

func TestConcurrentProcessesKeepEveryLineWhole(t *testing.T) {
	file := useCache(t)
	const workers = 8
	runWorkers(t, workers)
	if got := wholeLines(t, file); got != workers*workerLines {
		t.Fatalf("got %d lines, want %d", got, workers*workerLines)
	}
}

// A roll can lose to another process that has the file open, so this counts nothing: it
// holds only that no line breaks across one.
func TestConcurrentProcessesKeepLinesWholeAcrossARoll(t *testing.T) {
	file := useCache(t)
	filler := strings.Repeat("filler\n", (logline.Limit-4096)/len("filler\n"))
	if err := os.WriteFile(file, []byte(filler), 0o600); err != nil {
		t.Fatal(err)
	}
	runWorkers(t, 4)
	wholeLines(t, file)
	wholeLines(t, file+".old")
}

func crash(t *testing.T, kind string) (log, stderr string) {
	t.Helper()
	file := useCache(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe)
	cmd.Env = append(os.Environ(), crashEnv+"="+kind)
	var out strings.Builder
	cmd.Stderr = &out
	var exit *exec.ExitError
	if err := cmd.Run(); !errors.As(err, &exit) || exit.ExitCode() != 2 {
		t.Fatalf("want a crash's exit 2, got %v", err)
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(b), out.String()
}

func TestACrashLogsTheRuntimeErrorAndWhereThenDies(t *testing.T) {
	log, stderr := crash(t, "index")
	e := entry(t, log)
	if e.Message != "crash" || !strings.Contains(field(e, "error"), "runtime error: index out of range") ||
		!strings.Contains(field(e, "stack"), "faillog.TestMain:") {
		t.Fatalf("%+v", e)
	}
	if !strings.Contains(stderr, "index out of range") {
		t.Fatalf("the crash left stderr: %s", stderr)
	}
}

func TestACrashKeepsAnotherPanicsValueOutOfTheLog(t *testing.T) {
	log, stderr := crash(t, "value")
	if strings.Contains(log, "picked-item") {
		t.Fatalf("panic value in the log: %s", log)
	}
	if e := entry(t, log); e.Message != "crash" || field(e, "error") != "string" {
		t.Fatalf("%+v", e)
	}
	if !strings.Contains(stderr, "picked-item") {
		t.Fatalf("the crash left stderr: %s", stderr)
	}
}
