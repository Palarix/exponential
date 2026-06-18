package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/palarix/exponential/internal/demo"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var demoForceFlag bool

var demoCmd = &cobra.Command{
	Use:   "demo <output-directory>",
	Short: "Generate a demo project with realistic sample data",
	Long:  `Generates a fully populated .xpo directory with a fictional project (Upurr Eats — food delivery for cats) containing realistic issues, comments, status transitions, and history.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputDir := args[0]

		dbPath := filepath.Join(outputDir, ".xpo", "issues.db")
		if _, err := os.Stat(dbPath); err == nil && !demoForceFlag {
			fmt.Printf("%s %s already contains an issues.db.\n", ui.ErrorPrefix, outputDir)
			fmt.Println("Use --force to overwrite.")
			os.Exit(1)
		}

		if err := demo.Generate(outputDir); err != nil {
			fmt.Printf("%s Failed to generate demo project: %v\n", ui.ErrorPrefix, err)
			os.Exit(1)
		}

		fmt.Printf("%s Generated demo project in %s\n", ui.OKPrefix, filepath.Join(outputDir, ".xpo"))
		fmt.Println()
		fmt.Println("  To explore:")
		fmt.Printf("    cd %s && xpo list\n", outputDir)
		fmt.Printf("    cd %s && xpo board\n", outputDir)
	},
}

func init() {
	demoCmd.Flags().BoolVar(&demoForceFlag, "force", false, "Overwrite existing issues.db")
	rootCmd.AddCommand(demoCmd)
}
