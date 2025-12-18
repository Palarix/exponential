package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "beats",
	Short: "A JSONL-based issue tracker",
	Long:  `Beats is a JSONL-based issue tracker that is committed to Git together with your project.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
