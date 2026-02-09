package main

import (
	"fmt"
	"runtime"

	"github.com/palarix/beats/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of beats",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Beats Version: %s\n", version.CLIVersion)
		fmt.Printf("Data Model Version: %d\n", version.DataModelVersion)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
