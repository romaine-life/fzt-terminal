package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The release suite stamps main.version (pipeline-templates suites/go/build.mjs); --version
// prints it as render.Version.
func TestVersionPrintsTheStampedRelease(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fzt.exe")
	if out, err := exec.Command("go", "build", "-ldflags", "-X main.version=v0.1.999", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "v0.1.999" {
		t.Fatalf("--version printed %q", got)
	}
}
