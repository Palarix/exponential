package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/server"
	"github.com/palarix/exponential/internal/storage"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
)

var pulseJSON bool

var pulseCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Show project health metrics",
	Long: `Prints a compact summary of project tempo, cadence, and health.

Reads directly from the local event log — no server needed.`,
	RunE: runPulse,
}

func init() {
	pulseCmd.Flags().BoolVar(&pulseJSON, "json", false, "output raw metrics as JSON")
	rootCmd.AddCommand(pulseCmd)
}

func runPulse(cmd *cobra.Command, args []string) error {
	events, err := storage.ReadEvents()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	issues := exponential.ProjectIssuesWithConfig(events, cfg)
	m := server.ComputePulseMetrics(issues, time.Now())

	if pulseJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(m)
	}

	data := ui.PulseData{
		ProjectName:     cfg.Name,
		VelocityPts:     m.Velocity.CurrentWeekPoints + m.Velocity.Last7dPoints,
		VelocityDelta:   m.Velocity.Delta,
		CycleTimeHrs:    m.Flow.CycleTimeHrs,
		CycleCount:      m.Flow.CycleCount,
		LeadTimeHrs:     m.Flow.LeadTimeHrs,
		LeadCount:       m.Flow.LeadCount,
		WIPTotal:        m.WIP.Total,
		WIPStale:        m.WIP.Stale,
		BlockersTotal:   m.Blockers.Total,
		BlockersOldest:  m.Blockers.OldestDays,
		ThroughputWk:    m.Throughput.Last7d,
		ThroughputDelta: m.Throughput.Delta,
	}

	width := ui.TerminalWidth()
	fmt.Print(ui.RenderPulse(data, width))
	return nil
}
