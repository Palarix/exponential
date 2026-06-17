package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/palarix/exponential/internal/exponential"
	"github.com/palarix/exponential/internal/config"
	"github.com/palarix/exponential/internal/model"
	"github.com/palarix/exponential/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	cycleInitDuration string
	cycleInitStartDay string
	cycleInitAnchor   string
	cycleHistoryCount int
)

var cycleCmd = &cobra.Command{
	Use:   "cycle [current|next]",
	Short: "Show cycle info and issues",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !cfg.Cycles.Enabled {
			fmt.Println("Cycles are not configured. Run 'xpo cycle init' to set up.")
			os.Exit(1)
		}

		now := time.Now()
		var target config.Cycle

		sub := "current"
		if len(args) > 0 {
			sub = args[0]
		}

		switch sub {
		case "current":
			target = cfg.Cycles.CurrentCycle()
		case "next":
			cycles := cfg.Cycles.EnumerateCycles(now, 0, 1)
			target = cycles[1]
		default:
			c, err := cfg.Cycles.CycleForID(sub)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			target = c
		}

		printCycleWithIssues(target, cfg)
	},
}

var cycleInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Configure cycle cadence",
	Run: func(cmd *cobra.Command, args []string) {
		cc := config.CycleConfig{
			Enabled:    true,
			Duration:   cycleInitDuration,
			StartDay:   cycleInitStartDay,
			AnchorDate: cycleInitAnchor,
		}
		if err := cc.Validate(); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if err := writeCycleConfig(cc); err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			os.Exit(1)
		}

		current := cc.CurrentCycle()
		fmt.Printf("Cycles configured: %s cadence starting on %ss\n", cc.Duration, cc.StartDay)
		fmt.Printf("Current cycle: %s (%s to %s)\n",
			current.ID,
			current.Start.Format("Jan 2"),
			current.End.Format("Jan 2"))
	},
}

var cycleHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show past cycles with completion stats",
	Run: func(cmd *cobra.Command, args []string) {
		if !cfg.Cycles.Enabled {
			fmt.Println("Cycles are not configured. Run 'xpo cycle init' to set up.")
			os.Exit(1)
		}

		client := exponential.NewClient(cfg)
		issues, err := client.ListIssues(exponential.FilterOptions{All: true})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		now := time.Now()
		cycles := cfg.Cycles.EnumerateCycles(now, cycleHistoryCount, 0)

		for _, c := range cycles {
			done, total := 0, 0
			for _, issue := range issues {
				eid := issue.EffectiveCycleID
				if issue.Status == model.StatusDone {
					eid = issue.CycleID
				}
				if eid == c.ID {
					total++
					if issue.Status == model.StatusDone {
						done++
					}
				}
			}

			label := ""
			current := cfg.Cycles.CurrentCycle()
			if c.ID == current.ID {
				label = " (current)"
			} else if c.Start.After(current.End) {
				label = " (upcoming)"
			}

			fmt.Printf("Cycle %d  %s to %s  %d/%d done%s\n",
				c.Number,
				c.Start.Format("Jan 2"),
				c.End.Format("Jan 2"),
				done, total,
				label)
		}
	},
}

func printCycleWithIssues(c config.Cycle, cfg *config.Config) {
	current := cfg.Cycles.CurrentCycle()
	label := "Past"
	if c.ID == current.ID {
		label = "Current"
	} else if c.Start.After(current.End) {
		label = "Upcoming"
	}

	fmt.Printf("Cycle %d · %s · %s to %s\n\n",
		c.Number, label,
		c.Start.Format("Jan 2"),
		c.End.Format("Jan 2"))

	client := exponential.NewClient(cfg)
	issues, err := client.ListIssues(exponential.FilterOptions{
		CycleID: c.ID,
		All:     true,
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(issues) == 0 {
		fmt.Println("No issues in this cycle.")
		return
	}

	width := ui.TerminalWidth()
	fmt.Print(ui.RenderIssueList(issues, width))
}

func writeCycleConfig(cc config.CycleConfig) error {
	configPath := filepath.Join(".xpo", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		data = []byte{}
	}

	var doc yaml.Node
	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	if doc.Kind == 0 {
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}

	root := doc.Content[0]

	cyclesNode := &yaml.Node{Kind: yaml.MappingNode}
	cyclesNode.Content = append(cyclesNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "enabled"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "true"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "duration"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: cc.Duration},
		&yaml.Node{Kind: yaml.ScalarNode, Value: "start_day"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: cc.StartDay},
	)
	if cc.AnchorDate != "" {
		cyclesNode.Content = append(cyclesNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "anchor_date"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: cc.AnchorDate},
		)
	}

	found := false
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "cycles" {
			root.Content[i+1] = cyclesNode
			found = true
			break
		}
	}
	if !found {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "cycles"},
			cyclesNode,
		)
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

func resolveCycleID(value string) string {
	if !cfg.Cycles.Enabled {
		return value
	}
	now := time.Now()
	switch strings.ToLower(value) {
	case "current":
		return cfg.Cycles.CurrentCycle().ID
	case "next":
		cycles := cfg.Cycles.EnumerateCycles(now, 0, 1)
		return cycles[1].ID
	case "none", "":
		return ""
	default:
		return value
	}
}

func init() {
	cycleInitCmd.Flags().StringVar(&cycleInitDuration, "duration", "2w", "Cycle duration (1w, 2w, 3w, 4w)")
	cycleInitCmd.Flags().StringVar(&cycleInitStartDay, "start", "monday", "Day cycles start on")
	cycleInitCmd.Flags().StringVar(&cycleInitAnchor, "anchor", "", "Anchor date (YYYY-MM-DD) for first cycle")

	cycleHistoryCmd.Flags().IntVarP(&cycleHistoryCount, "count", "n", 5, "Number of past cycles to show")

	cycleCmd.AddCommand(cycleInitCmd)
	cycleCmd.AddCommand(cycleHistoryCmd)
	rootCmd.AddCommand(cycleCmd)
}
