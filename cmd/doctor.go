package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"dev-cli/internal/health"

	"github.com/spf13/cobra"
)

var (
	doctorFix   bool
	doctorQuiet bool
	doctorJSON  bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and fix issues",
	Long: `Run health checks on all dev-cli dependencies and optionally fix issues.

Checks:
  - Docker daemon status
  - Ollama availability  
  - GPU/CUDA support
  - Required directories
  - Network connectivity`,
	Example: `  # Run health checks
  dev-cli doctor

  # Auto-fix issues where possible
  dev-cli doctor --fix`,
	Run: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Attempt to auto-fix issues")
	doctorCmd.Flags().BoolVar(&doctorQuiet, "quiet", false, "Only show failures")
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON for agent consumption")
}

func runDoctor(cmd *cobra.Command, args []string) {
	checks := health.AllChecks()

	var failed, warned, passed int
	var jsonResults []health.CheckResultJSON

	for _, check := range checks {
		result := check()

		if doctorJSON {
			jsonResults = append(jsonResults, health.CheckResultJSON{
				Name:    result.Name,
				Status:  result.Status,
				Message: result.Message,
				FixCmd:  result.FixCmd,
			})
		}

		switch result.Status {
		case "ok":
			passed++
		case "warn":
			warned++
		case "fail":
			failed++
		}

		if doctorJSON {
			continue
		}

		if doctorQuiet && result.Status == "ok" {
			continue
		}

		icon := statusIcon(result.Status)
		fmt.Printf("%s %s\n", icon, fmtBold(result.Name))
		fmt.Printf("   %s\n", result.Message)

		if result.Status == "fail" {
			if doctorFix && (result.FixCmd != "" || result.FixFunc != nil) {
				fmt.Printf("   %s Attempting fix...%s\n", colorYellow, colorReset)
				if err := health.AttemptFix(result); err != nil {
					fmt.Printf("   %s Fix failed: %v%s\n", colorRed, err, colorReset)
				} else {
					printSuccess("Fixed!")
					failed--
					passed++
				}
			} else if result.FixCmd != "" {
				fmt.Printf("   %sFix: %s%s\n", colorCyan, result.FixCmd, colorReset)
			}
		}
		fmt.Println()
	}

	if doctorJSON {
		report := health.Report{
			Timestamp: time.Now().Format(time.RFC3339),
			Checks:    jsonResults,
			Summary: health.Summary{
				Passed: passed,
				Warned: warned,
				Failed: failed,
				Total:  len(checks),
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		if failed > 0 {
			os.Exit(1)
		}
		return
	}

	fmt.Println(fmtBold("dev-cli doctor"))
	separator()
	fmt.Printf("%s %d passed  ", iconOK(), passed)
	if warned > 0 {
		fmt.Printf("%s %d warnings  ", iconWarn(), warned)
	}
	if failed > 0 {
		fmt.Printf("%s%s %d failed%s", colorRed, iconFail(), failed, colorReset)
	}
	fmt.Println()

	if failed > 0 && !doctorFix {
		fmt.Printf("\n%sRun 'dev-cli doctor --fix' to attempt auto-fixes%s\n", colorYellow, colorReset)
		os.Exit(1)
	}
}
