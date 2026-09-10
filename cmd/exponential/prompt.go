package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/palarix/exponential/internal/ui"
	"golang.org/x/term"
)

// abortInit prints a friendly decline message and exits.
// hint is a short command suggestion like "git init && xpo init".
func abortInit(hint string) {
	fmt.Printf("\nNo changes were made.")
	if hint != "" {
		fmt.Printf(" When you're ready:\n  %s", hint)
	} else {
		fmt.Printf(" Run xpo init again when you're ready.")
	}
	fmt.Printf("\n\n")
	os.Exit(1)
}

// withCtrlC installs a SIGINT handler that prints the abort message and exits 130.
// Returns a cleanup function to restore the default handler.
func withCtrlC(hint string) func() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println()
		fmt.Printf("\nNo changes were made.")
		if hint != "" {
			fmt.Printf(" When you're ready:\n  %s", hint)
		} else {
			fmt.Printf(" Run xpo init again when you're ready.")
		}
		fmt.Printf("\n\n")
		os.Exit(130)
	}()
	return func() { signal.Stop(c) }
}

// promptConfirm shows an inline "? Question (Y/n)" prompt and returns the answer.
// Returns false on EOF (pipe closed / Ctrl-D).
func promptConfirm(question string, defaultYes bool) bool {
	hint := "(Y/n)"
	if !defaultYes {
		hint = "(y/N)"
	}
	fmt.Printf("\n%s %s %s ", ui.AccentStyle.Render("?"), question, ui.MutedStyle.Render(hint))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err == io.EOF {
		fmt.Println()
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultYes
	}
	return response == "y" || response == "yes"
}

// promptInput shows a live-updating input prompt.
// The value appears inline after › and a preview line below updates as the user types.
// previewFn takes the current value and returns the muted preview text.
func promptInput(label string, defaultVal string, previewFn func(string) string) string {
	if !isInteractive() {
		return defaultVal
	}

	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// Fallback to simple prompt
		fmt.Printf("\n%s %s %s ", ui.AccentStyle.Render("?"), label, ui.MutedStyle.Render("›"))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		if response == "" {
			return defaultVal
		}
		return response
	}
	defer term.Restore(fd, oldState)

	value := defaultVal
	buf := make([]byte, 1)

	hasPreview := previewFn != nil
	inputLine := func() string {
		return fmt.Sprintf("%s %s %s %s", ui.AccentStyle.Render("?"), label, ui.MutedStyle.Render("›"), value)
	}

	render := func() {
		// Clear and redraw input line
		fmt.Printf("\r\033[K%s", inputLine())
		if hasPreview {
			// Draw preview on line below, then move back up and reprint input
			// so cursor naturally lands at end of value
			fmt.Printf("\r\n\033[K  %s", ui.MutedStyle.Render(previewFn(value)))
			fmt.Printf("\033[A\r\033[K%s", inputLine())
		}
	}

	// Reserve two lines (input + preview) before starting
	fmt.Printf("\r\n")
	if hasPreview {
		fmt.Printf("\r\n")
	}
	// Move up to input line position
	if hasPreview {
		fmt.Printf("\033[A")
	}
	render()

	for {
		_, err := os.Stdin.Read(buf)
		if err != nil {
			break
		}
		b := buf[0]

		switch {
		case b == '\r' || b == '\n':
			if hasPreview {
				fmt.Printf("\r\n") // move past preview line
			}
			fmt.Printf("\r\n")
			return value
		case b == 3: // Ctrl-C
			if hasPreview {
				fmt.Printf("\r\n")
			}
			fmt.Printf("\r\n")
			term.Restore(fd, oldState)
			abortInit("")
			return defaultVal
		case b == 127 || b == 8: // Backspace
			if len(value) > 0 {
				value = value[:len(value)-1]
				render()
			}
		case b == 21: // Ctrl-U: clear input
			value = ""
			render()
		case b >= 32 && b < 127: // Printable ASCII
			value += string(b)
			render()
		}
	}

	fmt.Printf("\n")
	return value
}

// promptSelect shows an interactive select list with arrow key navigation.
func promptSelect(question string, options []string) int {
	var huhOpts []huh.Option[int]
	for i, opt := range options {
		huhOpts = append(huhOpts, huh.NewOption(opt, i))
	}

	var selected int
	huh.NewSelect[int]().
		Title(question).
		Options(huhOpts...).
		Value(&selected).
		Run()

	return selected
}
