package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// isStdinPiped reports whether stdin is connected to a pipe or redirected file
// (i.e. not a terminal). When false, reading from stdin would block waiting
// for interactive input.
func isStdinPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// readAllStdin reads stdin to EOF and strips trailing newlines. The rest of
// the content (including leading whitespace and internal newlines) is
// preserved verbatim.
func readAllStdin() (string, error) {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("failed to read from stdin: %w", err)
	}
	return strings.TrimRight(string(b), "\n"), nil
}

// readStdinExplicit reads stdin and errors if stdin is a terminal — used for
// the `-` opt-in form, where hanging waiting for keyboard input would be a
// poor experience.
func readStdinExplicit() (string, error) {
	if !isStdinPiped() {
		return "", fmt.Errorf("'-' was used to read from stdin, but stdin is a terminal")
	}
	return readAllStdin()
}
