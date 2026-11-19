//go:build !windows

package main

import "errors"

// paneSupported is false outside Windows, where fzt can't wait for a process
// that isn't its child.
const paneSupported = false

func waitForProcess(pid int) error {
	return errors.New("--pane works on Windows only")
}

func tmuxProgram(argv []string) []string { return argv }
