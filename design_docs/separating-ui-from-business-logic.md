# Separating UI from Business Logic

## Problem
Currently, we have business logic and presentation (UI) mixed in our commands.

## Proposed Solution
Refactor the application so we have a `UI` Package which is fully concerned with the presentation layer,
offering methods like: `Warn`, `Alert`, `Error`, `Success`, `Code`, different `Icons`, and covenience methods to `Format` strings. 
Also components like `Table`, `Checklist`, etc..

Example:

```go

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/samber/lo"
	"golang.org/x/term"
)

// Table returns a lipgloss table with the given headers and rows.
func Table(headers []string, rows [][]string) *table.Table {
	// Get the terminal width
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80
	}

	// Find the widest cell in each column
	colWidths := lo.Map(headers, headerWidth)
	for _, row := range rows {
		for i, cell := range row {
			if lipgloss.Width(cell) > colWidths[i] {
				colWidths[i] = lipgloss.Width(cell)
			}
		}
	}

	// Find the widest column
	maxCol := 0
	for i, width := range colWidths {
		if width > colWidths[maxCol] {
			maxCol = i
		}
	}

	// If the table is too wide, shrink the widest column
	totalWidth := lo.Sum(colWidths) + len(headers) - 1
	if totalWidth > termWidth {
		colWidths[maxCol] -= totalWidth - termWidth
	}

	return table.New().
		Border(lipgloss.HiddenBorder()).
		BorderHeader(false).
		BorderLeft(false).BorderRight(false).
		BorderTop(false).BorderBottom(false).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle()
			if row == table.HeaderRow {
				return style.Bold(true)
			}
			return style.Width(colWidths[col])
		})
}

func headerWidth(header string, _ int) int {
	return lipgloss.Width(header)
}
```

Also introduce separate `Beats` package that contains all the actual business logic (creating entries, deleting, updating, projecting, etc.)

Then commands become simple wrappers around business logic and presentation. Right now, each command is a separate self-container "snowflake", without unification. It makes the app look sloppy.

## Business Logic

### 1. The `beats init` command
- initializes the `.beats` folder
- checks if the folder already exists, exits with an `Error` if it does
- overwrites the folder and notifies with a `Warn` if it does exist but `--force` flag was given
- runs the checks logic for `beats doctor`, if the resulting `issues[]` array is non-empty shows a list of found issues and prompts the user to run `beats doctor`
- prints a summary at the end to indicate whether the initialization was successful or not

### 2. The `beats add` command
- Allows users to add a new issue
- First argument is the issue title, additional fields (description, type, parent, etc.) can be set with flags
- If no title or description given and terminal is interactive, open default editor with template
- If no title or description given and terminal is not interactive, exit with `Error`
- After interactive editor, require as a minimum the title; if no title given exist with `Error`
- After interactive editor, parse template, create new issues with title and description, use any flags for metadata
- After successful creation show a summary (Create id + title, below metadata, skip description text)

### 3. The `beats update` command
- Allows users to update the title, description or metadata for a given issue id
- If no arguments (other than id) or flags given, opens the default editor with an update template (if the terminal is interactive)
- If the editor closes without changes, `Warn` that issue was not updated since nothing changed
- If successful shows a summary message ("Updated title, descsription", or "Changed status to DOING", etc.)

### 4. The `beats show` command
- Allows users to show issue details, given the id of the issue
- renders first a header with id and title
- followed by metadata table
- followed by description rendered as markdown
- then a table of all the children issues (if any)
- then a table of all the blocking/blocked by issues (if any)
- then a table for the issue history (all events)
- if a given issue id is not found in the hot cache (`issues.db`) also searches the archive

### 5. The `beats history` command
- Shows a list of all events for the given issue id
- renders first a header with id and title
- followed by the history / events table (same table as with `show` command)

### 6. The `beats ls` command
- Shows a list of all open issues + the 3 most recent done issues
- List is rendered as a table with issue id, type, state, title, created by, created (date)
- groups issues by parent id
- sorts by created (date) in ascending order
- can be given --all or -a flag to show all DONE issues
- can be given --archived flag to include issues from the archive (by default only renders issues from hot cache `issues.db`)

### 7. The `beats delete` command
- Allows users to delete an issue by given id
- If interactive terminal prompt user for confirmation
- If noninteractive fail
- If --force flag given, execute regardless of interactive, skip prompt
- Show summary message when successful ("Deleted <kind> <id>")

### 8. The `beats doctor` command
- Performs a check of the beats setup:
- `./.beats` directory exists
- `./.beats/issues.db` valid (if exists)
- `./.git` directory exists, if exists, check if entry for beats local snapshot file is in `.gitignore`
- if `./.github/workflows` directory exists, check if beats exceptions added to the workflows (so we don't trigger builds everytime an issue is added)
- if supported shell, check if beats completions entry is set, otherwise prompt to set completions up
- Check for existence of a known AI agent file, and whether beats instructions are present in the agent file.
- For any issue flagged, show a `Warn` checklist 
- For any issue passed, show a `Success` checklist entry

Example:

```
beats doctor - Checking configuration...

[✔] Git repository detected
[✔] Beats database exists
[✔] Agent files with beats config:
    • Generic Agent (AGENTS.md)
    • Gemini (GEMINI.md)

[✔] Shell completion for zsh is configured

You are ready to use beats!
```

### 9. Shortcut commands for covenience
- Use `beats blocked` to mark a given issue as BLOCKED, must be given `--by <parent_id>` and optionally a `--reason`
- Use `beats done` to mark a given issue as DONE
- Use `beats planned` to mark a given issue as PLANNED
- Use `beats start` to mark a given issue as DOING
- Use `beats estimate` to add or modify the story point estimate for a given issue. If issue already has an estimate, prompt (if interactive), or skip propmt if --force
- Use `beats burned` to add the given story points to the burndown total for the given issue. `Warn` if the total exceeds the estimate.

### 10. Maintenance commands
- We can use the `beats archive` command to clean up the hot cache `issues.db` by moving older DONE issues to an archive and removing DELETED issues.
- We can use the `beats config show` command to show the current config (merged from all config sources)
- We can use the `beats config set` command to set a configuration option (key + value). Use `--local` flag to store in `./.beats/config` file instead of globally

### Completions

All commands that require an issue id show support autocompletion for issues (same list and sorting as `beats ls`), format is `<id> - <kind> <state> <title>`

## Structure

We want to logically separate commands, UI and business logic. For example we should have the different `beats doctor` checks under a `doctor/checks` directory rather than inside the `cmds` directory.