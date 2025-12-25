package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"dev-cli/internal/analytics"
	"dev-cli/internal/storage"

	"github.com/spf13/cobra"
)

var (
	analyticsJSON    bool
	analyticsLimit   int
	analyticsCommand string
	analyticsCluster bool
)

var analyticsCmd = &cobra.Command{
	Use:     "analytics",
	Short:   "View proactive debugging insights from command history",
	Aliases: []string{"stats", "insights"},
	Example: `  dev-cli analytics
  dev-cli analytics --command npm
  dev-cli analytics --cluster
  dev-cli analytics --json`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.InitDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to open db: %v\n", err)
			return
		}
		defer db.Close()

		analyzer := analytics.NewAnalyzer(db)

		if analyticsCluster {
			showErrorClusters(analyzer)
		} else if analyticsCommand != "" {
			showCommandStats(analyzer, analyticsCommand)
		} else {
			showFailurePatterns(analyzer)
		}
	},
}

func init() {
	rootCmd.AddCommand(analyticsCmd)
	analyticsCmd.Flags().BoolVar(&analyticsJSON, "json", false, "Output as JSON")
	analyticsCmd.Flags().IntVarP(&analyticsLimit, "limit", "l", 10, "Number of patterns to show")
	analyticsCmd.Flags().StringVarP(&analyticsCommand, "command", "c", "", "Analyze a specific command pattern")
	analyticsCmd.Flags().BoolVar(&analyticsCluster, "cluster", false, "Show error clusters using simhash")
}

func showCommandStats(analyzer *analytics.Analyzer, command string) {
	stats, err := analyzer.GetStats(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to get stats: %v\n", err)
		return
	}

	if analyticsJSON {
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("\n📊 Statistics for '%s'\n", stats.Pattern)
	fmt.Printf("   Total runs:    %d\n", stats.RunCount)
	fmt.Printf("   Failures:      %d\n", stats.FailCount)
	fmt.Printf("   Failure rate:  %.1f%%\n", stats.FailRate*100)
	fmt.Printf("   Avg duration:  %dms\n", stats.AvgDurationMs)

	if len(stats.KnownFixes) > 0 {
		fmt.Println("\n💡 Known Fixes:")
		for i, fix := range stats.KnownFixes {
			fmt.Printf("   %d. %s\n", i+1, fix)
		}
	}

	suggestions := analyzer.GetSuggestions(command)
	if len(suggestions) > 0 {
		fmt.Println("\n🔍 Suggestions:")
		for _, sug := range suggestions {
			icon := "💡"
			if sug.Priority == "high" {
				icon = "⚠️"
			} else if sug.Priority == "low" {
				icon = "ℹ️"
			}
			fmt.Printf("   %s %s\n", icon, sug.Description)
			if sug.Fix != "" {
				fmt.Printf("      → %s\n", sug.Fix)
			}
		}
	}
}

func showFailurePatterns(analyzer *analytics.Analyzer) {
	patterns, err := analyzer.GetFailurePatterns(analyticsLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to get patterns: %v\n", err)
		return
	}

	if len(patterns) == 0 {
		fmt.Println("✨ No failure patterns found in recent history")
		return
	}

	if analyticsJSON {
		data, _ := json.MarshalIndent(patterns, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("\n📊 Recent Failure Patterns (%d found)\n\n", len(patterns))
	for i, p := range patterns {
		rateColor := "\033[32m"
		if p.FailRate > 0.5 {
			rateColor = "\033[31m"
		} else if p.FailRate > 0.25 {
			rateColor = "\033[33m"
		}
		fmt.Printf("  %d. %s\n", i+1, p.Pattern)
		fmt.Printf("     Failures: %d | Rate: %s%.0f%%\033[0m\n",
			p.FailCount, rateColor, p.FailRate*100)
	}

	fmt.Println("\n💡 Use --command <name> or --cluster for more details")
}

func showErrorClusters(analyzer *analytics.Analyzer) {
	clusters, err := analyzer.ClusterErrors(analyticsLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to cluster errors: %v\n", err)
		return
	}

	if len(clusters) == 0 {
		fmt.Println("✨ No error clusters found")
		return
	}

	if analyticsJSON {
		data, _ := json.MarshalIndent(clusters, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("\n🔗 Error Clusters (%d found)\n\n", len(clusters))
	for i, c := range clusters {
		fmt.Printf("  %d. %s (x%d)\n", i+1, c.Pattern, c.Count)
		fmt.Printf("     Fingerprint: %x\n", c.Fingerprint)
		if len(c.Samples) > 0 {
			fmt.Printf("     Sample: %s\n", c.Samples[0])
		}
		if len(c.Solutions) > 0 {
			fmt.Printf("     💡 Fix: %s\n", c.Solutions[0])
		}
	}
}
