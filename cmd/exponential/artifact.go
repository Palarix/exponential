package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var (
	artifactSpecFlag        bool
	artifactWalkthroughFlag bool
	artifactNameFlag        string
	artifactFileFlag        string
)

var artifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Manage issue artifacts (specs, walkthroughs, attachments)",
}

var artifactAddCmd = &cobra.Command{
	Use:               "add <issue-id>",
	Short:             "Add or update an artifact on an issue",
	Long:              "Write an artifact to an issue. Use --spec or --walkthrough for first-class artifacts, or --name for generic files. Content is read from --file or piped stdin.",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		filename, artifactType, err := resolveAddFilename(artifactSpecFlag, artifactWalkthroughFlag, artifactNameFlag)
		if err != nil {
			return err
		}

		content, err := readArtifactContent(artifactFileFlag)
		if err != nil {
			return err
		}

		client := exponential.NewClient(cfg)

		switch artifactType {
		case "spec":
			err = client.WriteSpec(issueID, content)
		case "walkthrough":
			err = client.WriteWalkthrough(issueID, content)
		default:
			err = client.AddArtifact(issueID, "generic", filename, content)
		}
		if err != nil {
			return err
		}

		fmt.Printf("Artifact %s written to %s\n", filename, issueID)
		return nil
	},
}

var artifactShowCmd = &cobra.Command{
	Use:               "show <issue-id> [filename]",
	Short:             "Print artifact content to stdout",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		var positional string
		if len(args) == 2 {
			positional = args[1]
		}
		filename, err := resolveReadFilename(positional, artifactSpecFlag, artifactWalkthroughFlag)
		if err != nil {
			return err
		}

		client := exponential.NewClient(cfg)
		content, err := client.ReadArtifact(issueID, filename)
		if err != nil {
			return err
		}

		fmt.Print(content)
		return nil
	},
}

var artifactDeleteCmd = &cobra.Command{
	Use:               "delete <issue-id> [filename]",
	Short:             "Remove an artifact from an issue",
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		var positional string
		if len(args) == 2 {
			positional = args[1]
		}
		filename, err := resolveReadFilename(positional, artifactSpecFlag, artifactWalkthroughFlag)
		if err != nil {
			return err
		}

		client := exponential.NewClient(cfg)
		if err := client.DeleteArtifact(issueID, filename); err != nil {
			return err
		}

		fmt.Printf("Artifact %s deleted from %s\n", filename, issueID)
		return nil
	},
}

var artifactListCmd = &cobra.Command{
	Use:               "list <issue-id>",
	Short:             "List artifacts attached to an issue",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeIssueIDs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issueID := args[0]

		client := exponential.NewClient(cfg)
		artifacts, err := client.ListArtifacts(issueID)
		if err != nil {
			return err
		}

		if len(artifacts) == 0 {
			fmt.Printf("No artifacts on %s\n", issueID)
			return nil
		}

		termWidth := ui.TerminalWidth()

		typeWidth := 12
		filenameWidth := 24
		updatedWidth := 10
		padding := 3
		updatedByWidth := termWidth - typeWidth - filenameWidth - updatedWidth - padding
		if updatedByWidth < 12 {
			updatedByWidth = 12
		}

		header := fmt.Sprintf("%s %s %s %s",
			ui.RenderCell("TYPE", typeWidth, ui.MutedStyle),
			ui.RenderCell("FILENAME", filenameWidth, ui.MutedStyle),
			ui.RenderCell("UPDATED", updatedWidth, ui.MutedStyle),
			ui.RenderCell("UPDATED BY", updatedByWidth, ui.MutedStyle),
		)
		fmt.Println(header)
		fmt.Println(ui.MutedStyle.Render(strings.Repeat("─", termWidth)))

		for _, a := range artifacts {
			timeStr := humanize.Time(a.UpdatedAt)
			fmt.Printf("%s %s %s %s\n",
				ui.RenderCell(a.ArtifactType, typeWidth, ui.AccentStyle),
				ui.RenderCell(a.Filename, filenameWidth, ui.WhiteStyle),
				ui.RenderCell(timeStr, updatedWidth, ui.MutedStyle),
				ui.RenderCell(a.UpdatedBy, updatedByWidth, ui.MutedStyle),
			)
		}

		return nil
	},
}

func resolveAddFilename(spec, walkthrough bool, name string) (filename, artifactType string, err error) {
	set := 0
	if spec {
		set++
	}
	if walkthrough {
		set++
	}
	if name != "" {
		set++
	}

	if set == 0 {
		return "", "", fmt.Errorf("--name is required for generic artifacts (or use --spec / --walkthrough)")
	}
	if set > 1 {
		return "", "", fmt.Errorf("--spec, --walkthrough, and --name are mutually exclusive")
	}

	switch {
	case spec:
		return "spec.md", "spec", nil
	case walkthrough:
		return "walkthrough.md", "walkthrough", nil
	default:
		return name, "generic", nil
	}
}

func resolveReadFilename(positional string, spec, walkthrough bool) (string, error) {
	sources := 0
	if positional != "" {
		sources++
	}
	if spec {
		sources++
	}
	if walkthrough {
		sources++
	}

	if sources == 0 {
		return "", fmt.Errorf("filename is required (positional arg, --spec, or --walkthrough)")
	}
	if sources > 1 {
		return "", fmt.Errorf("provide exactly one of: positional filename, --spec, or --walkthrough")
	}

	switch {
	case spec:
		return "spec.md", nil
	case walkthrough:
		return "walkthrough.md", nil
	default:
		return positional, nil
	}
}

func readArtifactContent(filePath string) (string, error) {
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %q: %w", filePath, err)
		}
		return string(data), nil
	}

	if isStdinPiped() {
		return readAllStdin()
	}

	return "", fmt.Errorf("content is required: use --file <path> or pipe to stdin")
}

func init() {
	artifactAddCmd.Flags().BoolVar(&artifactSpecFlag, "spec", false, "Write as spec (spec.md)")
	artifactAddCmd.Flags().BoolVar(&artifactWalkthroughFlag, "walkthrough", false, "Write as walkthrough (walkthrough.md)")
	artifactAddCmd.Flags().StringVar(&artifactNameFlag, "name", "", "Filename for generic artifacts")
	artifactAddCmd.Flags().StringVar(&artifactFileFlag, "file", "", "Read content from file")

	artifactShowCmd.Flags().BoolVar(&artifactSpecFlag, "spec", false, "Shorthand for spec.md")
	artifactShowCmd.Flags().BoolVar(&artifactWalkthroughFlag, "walkthrough", false, "Shorthand for walkthrough.md")

	artifactDeleteCmd.Flags().BoolVar(&artifactSpecFlag, "spec", false, "Shorthand for spec.md")
	artifactDeleteCmd.Flags().BoolVar(&artifactWalkthroughFlag, "walkthrough", false, "Shorthand for walkthrough.md")

	artifactCmd.AddCommand(artifactAddCmd)
	artifactCmd.AddCommand(artifactShowCmd)
	artifactCmd.AddCommand(artifactDeleteCmd)
	artifactCmd.AddCommand(artifactListCmd)
	rootCmd.AddCommand(artifactCmd)
}
