package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// paneSupported: Windows can wait for any process by its handle.
const paneSupported = true

// waitForProcess returns once process pid has ended. Windows lets any
// process wait on another's handle, so the split's end is an event, never a
// poll. A process already gone can't be opened, and counts as ended.
func waitForProcess(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	_, err = p.Wait()
	return err
}

// tmuxProgram is a program and its arguments as split-window takes them. The
// Windows tmux joins them with spaces into the command line it starts, so each
// is quoted the way CreateProcess splits them (fzt-automate's tmuxProgram).
func tmuxProgram(argv []string) []string {
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = windows.EscapeArg(a)
	}
	return out
}
