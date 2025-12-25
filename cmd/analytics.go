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
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "View proactive debugging insights from command history",
	Long: `Analyze your command history to identify failure patterns and get proactive suggestions.
Shows failure rates, common fixes, and debugging hints based on historical data.`,
	Example: `  # View recent failure patterns
  dev-cli analytics

  # Get stats for a specific command
  dev-cli analytics --command npm

  # Output as JSON
  dev-cli analytics --json`,
	Aliases: []string{"stats", "insights"},
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.InitDB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Failed to open db: %v\n", err)
			return
		}
		defer db.Close()

		analyzer := analytics.NewAnalyzer(db)

		if analyticsCommand != "" {
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
}

func showCommandStats(analyzer *analytics.Analyzer, command string) {
	stats, err := analyzer.GetCommandStats(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Failed to get stats: %v\n", err)
		return
	}

	if analyticsJSON {
		data, _ := json.MarshalIndent(stats, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Printf("\n📊 Statistics for '%s'\n", stats.CommandPattern)
	fmt.Printf("   Total runs:    %d\n", stats.TotalRuns)
	fmt.Printf("   Failures:      %d\n", stats.FailureCount)
	fmt.Printf("   Failure rate:  %.1f%%\n", stats.FailureRate*100)
	fmt.Printf("   Avg duration:  %dms\n", stats.AvgDurationMs)

	if len(stats.CommonFixes) > 0 {
		fmt.Println("\n💡 Known Fixes:")
		for i, fix := range stats.CommonFixes {
			fmt.Printf("   %d. %s\n", i+1, fix)
		}
	}

	// Show proactive suggestions
	suggestions := analyzer.GetProactiveSuggestions(command)
	if len(suggestions) > 0 {
		fmt.Println("\n🔍 Suggestions:")
		for _, sug := range suggestions {
			icon := "💡"
			if sug.Severity == "high" {
				icon = "⚠️"
			} else if sug.Severity == "low" {
				icon = "ℹ️"
			}
			fmt.Printf("   %s %s\n", icon, sug.Message)
			if sug.SuggestedFix != "" {
				fmt.Printf("      → %s\n", sug.SuggestedFix)
			}
		}
	}
}

func showFailurePatterns(analyzer *analytics.Analyzer) {
	patterns, err := analyzer.GetRecentFailurePatterns(analyticsLimit)
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
		rateColor := "\033[32m" // green
		if p.FailureRate > 0.5 {
			rateColor = "\033[31m" // red
		} else if p.FailureRate > 0.25 {
			rateColor = "\033[33m" // yellow
		}
		fmt.Printf("  %d. %s\n", i+1, p.CommandPattern)
		fmt.Printf("     Failures: %d | Rate: %s%.0f%%\033[0m\n",
			p.FailureCount, rateColor, p.FailureRate*100)
	}

	fmt.Println("\n💡 Use --command <name> for detailed stats on a specific command")
}
