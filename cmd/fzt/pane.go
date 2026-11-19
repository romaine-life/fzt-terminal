package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// --pane puts the picker in front of the person from a program that has no
// terminal of its own, such as a Claude session's shell: fzt opens a tmux
// split under the pane it runs in (TMUX_PANE), runs itself there with the
// same flags, waits for that process to end, and prints its answer and
// returns its exit code as if it had run here. The split reads the input and
// writes its answer, errors and exit code in a folder of its own (--pane-io).

// exit ends fzt. Under --pane-io it first records the code for the fzt that
// opened the split.
var exit = os.Exit

const (
	paneIn   = "in"
	paneOut  = "out"
	paneErr  = "err"
	paneCode = "code"
	// paneCaller holds the asking fzt's process ID.
	paneCaller = "caller"
)

// usePaneIO makes the split's fzt read its input from dir and write its
// answer, errors and exit code there.
func usePaneIO(dir string) error {
	in, err := os.Open(filepath.Join(dir, paneIn))
	if err != nil {
		return err
	}
	out, err := os.Create(filepath.Join(dir, paneOut))
	if err != nil {
		return err
	}
	errFile, err := os.Create(filepath.Join(dir, paneErr))
	if err != nil {
		return err
	}
	os.Stdin, os.Stdout, os.Stderr = in, out, errFile
	exit = func(code int) {
		out.Close()
		errFile.Close()
		os.WriteFile(filepath.Join(dir, paneCode), []byte(strconv.Itoa(code)), 0o600)
		os.Exit(code)
	}
	// The question belongs to the fzt that asked it: once that one has ended
	// (its caller stopped, its session restarted), nobody can read an answer,
	// so the split closes and takes the folder with it.
	if text, err := os.ReadFile(filepath.Join(dir, paneCaller)); err == nil {
		if pid, err := strconv.Atoi(string(text)); err == nil {
			go func() {
				if waitForProcess(pid) == nil {
					in.Close()
					out.Close()
					errFile.Close()
					os.RemoveAll(dir)
					os.Exit(130)
				}
			}()
		}
	}
	return nil
}

var paneFlag = regexp.MustCompile(`^--?pane(=.*)?$`)

// paneArgs are the split's flags: these, without --pane.
func paneArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if a == "--" {
			break
		}
		if !paneFlag.MatchString(a) {
			out = append(out, a)
		}
	}
	return out
}

// runInPane runs fzt with args in a split under TMUX_PANE, lines as its
// input (nil with --yaml, which the split loads from the same folder), and
// returns its exit code once the split's fzt has ended.
func runInPane(args []string, lines []string) int {
	if !paneSupported {
		fail("--pane works on Windows only", "--pane on an unsupported system")
	}
	target := os.Getenv("TMUX_PANE")
	if target == "" {
		fail("--pane runs fzt in a tmux split, and this isn't running in tmux (TMUX_PANE isn't set)", "--pane outside tmux")
	}
	self, err := os.Executable()
	if err != nil {
		fail("can't find fzt's own program: "+err.Error(), "--pane couldn't find its program")
	}
	dir, err := os.MkdirTemp("", "fzt-pane-")
	if err != nil {
		fail("can't make the --pane folder: "+err.Error(), "--pane couldn't make its folder")
	}
	code, shown, logged := splitAndWait(target, self, dir, args, lines)
	os.RemoveAll(dir)
	if shown != "" {
		fail(shown, logged)
	}
	return code
}

// splitAndWait does runInPane's work in dir, which the caller removes. A
// failure comes back as fail's two texts.
func splitAndWait(target, self, dir string, args, lines []string) (code int, shown, logged string) {
	input := strings.Join(lines, "\n")
	if err := os.WriteFile(filepath.Join(dir, paneIn), []byte(input), 0o600); err != nil {
		return 0, "can't write the --pane input: " + err.Error(), "--pane couldn't write its input"
	}
	if err := os.WriteFile(filepath.Join(dir, paneCaller), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		return 0, "can't write the --pane input: " + err.Error(), "--pane couldn't write its input"
	}
	split := []string{"split-window", "-v", "-t", target, "-l", strconv.Itoa(paneHeight(target, len(lines))),
		"-P", "-F", "#{pane_id} #{pane_pid}"}
	if cwd, err := os.Getwd(); err == nil {
		split = append(split, "-c", cwd)
	}
	program := append(append([]string{self}, args...), "--pane-io", dir)
	split = append(split, tmuxProgram(program)...)
	out, err := exec.Command("tmux", split...).Output()
	if err != nil {
		return 0, "tmux couldn't open the split: " + tmuxError(err), "--pane: tmux couldn't open the split"
	}
	var paneID string
	var pid int
	if _, err := fmt.Sscan(string(out), &paneID, &pid); err != nil {
		return 0, fmt.Sprintf("tmux didn't say which split it opened (%q)", strings.TrimSpace(string(out))), "--pane: no pane id from tmux"
	}
	// A tmux restore brings the split back as a shell, never as this
	// checklist, whose folder will be gone.
	exec.Command("tmux", "set-option", "-p", "-t", paneID, "@session-set-shell", "1").Run()

	if err := waitForProcess(pid); err != nil {
		return 0, "can't wait for the split's fzt: " + err.Error(), "--pane couldn't wait for its split"
	}

	answer, _ := os.ReadFile(filepath.Join(dir, paneOut))
	os.Stdout.Write(answer)
	errText, _ := os.ReadFile(filepath.Join(dir, paneErr))
	os.Stderr.Write(errText)
	text, err := os.ReadFile(filepath.Join(dir, paneCode))
	if err != nil {
		// Its pane was closed, or it died, before it could answer.
		return 0, "the split closed without an answer", "--pane: the split closed without an answer"
	}
	code, err = strconv.Atoi(string(text))
	if err != nil {
		return 0, fmt.Sprintf("the split's exit code %q isn't a number", text), "--pane: bad exit code"
	}
	return code, "", ""
}

// paneHeight fits the split to the list: a row per line plus the frame,
// between 8 rows and half the pane it splits.
func paneHeight(target string, rows int) int {
	h := rows + 7 // border, search box, padding
	out, err := exec.Command("tmux", "display-message", "-p", "-t", target, "#{pane_height}").Output()
	if err != nil {
		return max(h, 8)
	}
	total, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return max(h, 8)
	}
	return max(min(h, total/2), 8)
}

// tmuxError is tmux's own message for a failed command.
func tmuxError(err error) string {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
		return strings.TrimSpace(string(exitErr.Stderr))
	}
	return err.Error()
}
