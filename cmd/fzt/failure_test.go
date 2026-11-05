package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const runMainEnv = "FZT_TEST_RUN_MAIN"

// TestMain runs the real fzt when a test starts this binary with runMainEnv set, so a
// child never runs the tests.
func TestMain(m *testing.M) {
	if os.Getenv(runMainEnv) != "" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type run struct {
	exit           int
	stdout, stderr string
	log            string // the log file, empty when none was written
}

// fzt runs main in a child with its own log folder; prepare sets the child up further.
func fzt(t *testing.T, stdin string, prepare func(*exec.Cmd), args ...string) run {
	t.Helper()
	cache := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), runMainEnv+"=1", "LocalAppData="+cache, "XDG_CACHE_HOME="+cache)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if prepare != nil {
		prepare(cmd)
	}
	r := run{}
	var exit *exec.ExitError
	if err := cmd.Run(); errors.As(err, &exit) {
		r.exit = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	r.stdout, r.stderr = stdout.String(), stderr.String()
	if b, err := os.ReadFile(filepath.Join(cache, "fzt", "fzt.log")); err == nil {
		r.log = string(b)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return r
}

// failed checks what every failure shares: exit 2, nothing on stdout, and one line in
// the log holding want and none of the caller's text.
func failed(t *testing.T, r run, want string, private ...string) {
	t.Helper()
	if r.exit != 2 || r.stdout != "" {
		t.Fatalf("want exit 2 and no stdout: %+v", r)
	}
	if strings.Count(r.log, "\n") != 1 || !strings.Contains(r.log, want) {
		t.Fatalf("want one line holding %q: %q", want, r.log)
	}
	for _, text := range private {
		if strings.Contains(r.log, text) {
			t.Fatalf("%q reached the log: %q", text, r.log)
		}
	}
}

func TestRunsThatWentWellLogNothing(t *testing.T) {
	if r := fzt(t, "alpha\nbeta\n", nil, "--filter", "alp"); r.exit != 0 || r.stdout != "alpha\n" || r.log != "" {
		t.Fatalf("filter: %+v", r)
	}
	if r := fzt(t, "\n", nil); r.exit != 1 || r.log != "" {
		t.Fatalf("an empty list: %+v", r)
	}
	if r := fzt(t, "", nil, "-h"); r.exit != 0 || r.log != "" {
		t.Fatalf("help: %+v", r)
	}
}

func TestAMissingYAMLIsLoggedWithoutItsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private-name.yaml")
	r := fzt(t, "", nil, "--yaml", path, "--title", "private-title")
	failed(t, r, "couldn't load --yaml: ", "private-name", "private-title")
	if !strings.Contains(r.log, "(flags: title,yaml)") || !strings.Contains(r.stderr, "private-name") {
		t.Fatalf("%+v", r)
	}
}

func TestABadYAMLIsLoggedWithoutItsContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "menu.yaml")
	if err := os.WriteFile(path, []byte("private-item"), 0o600); err != nil {
		t.Fatal(err)
	}
	failed(t, fzt(t, "", nil, "--yaml", path), "couldn't load --yaml: not a valid menu", "private-item", "menu.yaml")
}

func TestNoInputIsLogged(t *testing.T) {
	failed(t, fzt(t, "", nil), "couldn't read input: no input")
}

func TestALineTooLongToReadIsLoggedWithoutIt(t *testing.T) {
	r := fzt(t, "private-item"+strings.Repeat("x", 1<<20)+"\n", nil)
	failed(t, r, "couldn't read input: bufio.Scanner: token too long", "private-item")
}

func TestAnUnknownFlagIsLoggedByName(t *testing.T) {
	r := fzt(t, "alpha\n", nil, "--no-such-flag")
	failed(t, r, "flag provided but not defined: -no-such-flag")
	if !strings.Contains(r.stderr, "no-such-flag") {
		t.Fatalf("%+v", r)
	}
}

func TestAFlagsValueStaysOutOfTheLog(t *testing.T) {
	failed(t, fzt(t, "alpha\n", nil, "--depth-penalty=private-text"), "a flag has a bad value", "private-text")
	failed(t, fzt(t, "alpha\n", nil, "-private item"), "a flag has a bad value", "private")
}
