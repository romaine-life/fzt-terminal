package faillog

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
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
			Write(fmt.Sprintf("worker %s line %d %s", id, i, strings.Repeat("x", 1000)))
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

var workerLine = regexp.MustCompile(`^\S+ \S+ \S+ pid \d+: worker \d+ line \d+ x{1000}$`)

func useCache(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LocalAppData", dir)
	t.Setenv("XDG_CACHE_HOME", dir)
	file := filepath.Join(dir, "fzt", "fzt.log")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestALineStartsWithItsTimeAndStaysOneLine(t *testing.T) {
	file := useCache(t)
	Write("first\r\nsecond")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if strings.Count(text, "\n") != 1 || !strings.HasSuffix(text, "\n") {
		t.Fatalf("not one line: %q", text)
	}
	stamp, _, _ := strings.Cut(text, " ")
	if _, err := time.Parse(time.RFC3339, stamp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "first  second") {
		t.Fatalf("%q", text)
	}
}

func TestPastTheLimitTheLogRolls(t *testing.T) {
	file := useCache(t)
	if err := os.WriteFile(file, make([]byte, limit+1), 0o600); err != nil {
		t.Fatal(err)
	}
	Write("after the roll")
	if info, err := os.Stat(file + ".old"); err != nil || info.Size() != limit+1 {
		t.Fatalf("old file: %v %v", info, err)
	}
	if info, err := os.Stat(file); err != nil || info.Size() > 200 {
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

// wholeLines fails on any line a concurrent write broke, and counts the workers' lines.
func wholeLines(t *testing.T, file string) int {
	t.Helper()
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines := bufio.NewScanner(f)
	lines.Buffer(make([]byte, limit), limit)
	seen := 0
	for lines.Scan() {
		switch text := lines.Text(); {
		case text == "filler":
		case workerLine.MatchString(text):
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
	filler := strings.Repeat("filler\n", (limit-4096)/len("filler\n"))
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
	for _, want := range []string{"crash: runtime error: index out of range", "faillog.TestMain:"} {
		if !strings.Contains(log, want) {
			t.Fatalf("log missing %q: %s", want, log)
		}
	}
	if strings.Count(log, "\n") != 1 {
		t.Fatalf("not one line: %s", log)
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
	if !strings.Contains(log, "crash: string at ") {
		t.Fatalf("%s", log)
	}
	if !strings.Contains(stderr, "picked-item") {
		t.Fatalf("the crash left stderr: %s", stderr)
	}
}
