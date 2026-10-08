package main

import (
	"runtime/debug"
	"testing"
)

// bubbletea v1's package init queries the terminal's background colour and
// waits up to 5s for a reply, stalling every command under a pty that doesn't
// answer (xpo-d053b5). The test binary links the same dependencies as the CLI.
func TestBinaryDoesNotLinkBubbleteaV1(t *testing.T) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Skip("no build info")
	}
	for _, dep := range info.Deps {
		if dep.Path == "github.com/charmbracelet/bubbletea" {
			t.Fatalf("bubbletea v1 (%s) is linked; its init() queries the terminal on every run", dep.Version)
		}
	}
}
