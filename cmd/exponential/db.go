package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/palarix/exponential/internal/storage"
	"github.com/palarix/exponential/internal/storage/refstore"
	"github.com/spf13/cobra"
)

var dbAuditCount int

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage the xpo event database",
	Long:  "Commands for inspecting, editing, and recovering the refs/xpo/data event store.",
}

var dbAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Show the event store commit history",
	Long:  "Lists every commit to refs/xpo/data — who changed what, when.",
	Run: func(cmd *cobra.Command, args []string) {
		store := requireRefStore()
		entries, err := store.Log(dbAuditCount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if len(entries) == 0 {
			fmt.Println("No history.")
			return
		}
		for _, e := range entries {
			sha := shortSHA(e.SHA)
			fmt.Printf("\033[33m%s\033[0m  %s\n", sha, e.Message)
			fmt.Printf("           %s  %s\n", e.Author, e.Date)
		}
	},
}

var dbEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit issues.db in $EDITOR",
	Long:  "Opens the event log in your editor. Changes are committed to refs/xpo/data on save.",
	Run: func(cmd *cobra.Command, args []string) {
		store := requireRefStore()

		content, err := store.ReadFile("issues.db")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading issues.db: %v\n", err)
			os.Exit(1)
		}

		tmpFile, err := os.CreateTemp("", "xpo-edit-*.db")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
			os.Exit(1)
		}
		tmpPath := tmpFile.Name()
		defer os.Remove(tmpPath)

		tmpFile.WriteString(content)
		tmpFile.Close()

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		editorCmd := exec.Command(editor, tmpPath)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr
		if err := editorCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Editor exited with error: %v\n", err)
			os.Exit(1)
		}

		edited, err := os.ReadFile(tmpPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading edited file: %v\n", err)
			os.Exit(1)
		}

		if string(edited) == content {
			fmt.Println("No changes.")
			return
		}

		oldLines := countNonEmpty(content)
		newLines := countNonEmpty(string(edited))

		fmt.Printf("Lines: %d → %d", oldLines, newLines)
		if newLines < oldLines {
			fmt.Printf("  (%d removed)", oldLines-newLines)
		} else if newLines > oldLines {
			fmt.Printf("  (%d added)", newLines-oldLines)
		}
		fmt.Println()

		if err := store.WriteFile("issues.db", string(edited), "xpo db edit: manual edit"); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing changes: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Saved to refs/xpo/data.")
	},
}

var dbRevertCmd = &cobra.Command{
	Use:   "revert <sha>",
	Short: "Revert a specific commit from the event store",
	Long:  "Restores issues.db to the state just before the given commit, then re-applies all commits after it. Use 'xpo db audit' to find the SHA.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		store := requireRefStore()
		targetSHA := args[0]

		parentSHA, err := resolveParent(store, targetSHA)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		beforeContent, err := store.ReadFileAt(parentSHA, "issues.db")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading state before %s: %v\n", shortSHA(targetSHA), err)
			os.Exit(1)
		}

		current, _ := store.ReadFile("issues.db")
		if beforeContent == current {
			fmt.Println("Nothing to revert — state is already equivalent.")
			return
		}

		msg := fmt.Sprintf("xpo db revert: undo %s", shortSHA(targetSHA))
		if err := store.WriteFile("issues.db", beforeContent, msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing reverted state: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Reverted commit %s. Event log restored to parent state.\n", shortSHA(targetSHA))
		fmt.Println("Use 'xpo db audit' to verify.")
	},
}

var dbResetCmd = &cobra.Command{
	Use:   "reset <sha>",
	Short: "Reset the event store to a previous commit",
	Long:  "Moves refs/xpo/data to point at the given SHA, discarding all commits after it. The discarded commits remain in git's object store until gc.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		store := requireRefStore()
		targetSHA := args[0]

		fullSHA, err := resolveCommit(store, targetSHA)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		currentSHA, _ := store.CurrentSHA()
		if fullSHA == currentSHA {
			fmt.Println("Already at that commit.")
			return
		}

		entries, _ := store.Log(0)
		var discardCount int
		for _, e := range entries {
			if e.SHA == fullSHA {
				break
			}
			discardCount++
		}

		fmt.Printf("This will discard %d commit(s) from the event store.\n", discardCount)
		fmt.Printf("Target: %s\n", shortSHA(fullSHA))
		fmt.Print("Proceed? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return
		}

		if err := store.ResetTo(fullSHA); err != nil {
			fmt.Fprintf(os.Stderr, "Error resetting: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Reset to %s. Discarded commits are recoverable via git reflog.\n", shortSHA(fullSHA))
	},
}

func requireRefStore() *refstore.Store {
	if !storage.RefStoreReady() {
		fmt.Fprintln(os.Stderr, "Error: ref-based storage (refs/xpo/data) is not initialized.")
		fmt.Fprintln(os.Stderr, "Run 'xpo init' to set up the project.")
		os.Exit(1)
	}
	return storage.RefStore()
}

func resolveParent(store *refstore.Store, sha string) (string, error) {
	out, err := store.ReadFileAt(sha+"^", "issues.db")
	_ = out
	if err != nil {
		return "", fmt.Errorf("cannot find parent of %s — it may be the initial commit", shortSHA(sha))
	}
	// Get the actual parent SHA
	entries, err := store.Log(0)
	if err != nil {
		return "", err
	}
	for i, e := range entries {
		if strings.HasPrefix(e.SHA, sha) || e.SHA == sha {
			if i+1 < len(entries) {
				return entries[i+1].SHA, nil
			}
			return "", fmt.Errorf("%s is the initial commit — cannot revert", shortSHA(sha))
		}
	}
	return "", fmt.Errorf("commit %s not found in refs/xpo/data history", sha)
}

func resolveCommit(store *refstore.Store, sha string) (string, error) {
	entries, err := store.Log(0)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.SHA, sha) || e.SHA == sha {
			return e.SHA, nil
		}
	}
	return "", fmt.Errorf("commit %s not found in refs/xpo/data history", sha)
}

func shortSHA(s string) string {
	if len(s) > 10 {
		return s[:10]
	}
	return s
}

func countNonEmpty(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

func init() {
	dbAuditCmd.Flags().IntVarP(&dbAuditCount, "count", "n", 0, "Number of entries to show (0 = all)")
	dbCmd.AddCommand(dbAuditCmd)
	dbCmd.AddCommand(dbEditCmd)
	dbCmd.AddCommand(dbRevertCmd)
	dbCmd.AddCommand(dbResetCmd)
	rootCmd.AddCommand(dbCmd)
}
