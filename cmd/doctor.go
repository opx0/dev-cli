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
	doctorQuiet bool
	doctorJSON  bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health and dependencies",
	Long: `Run read-only health checks on all dev-cli dependencies.

Checks:
  - Optional Docker daemon status
  - Local Ollama and configured model
  - Available LLM provider
  - Local data directory`,
	Example: `  dev-cli doctor
  dev-cli doctor --json`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorQuiet, "quiet", false, "Only show failures")
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON for agent consumption")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	checks := health.AllChecks()

	var failed, warned, passed int
	var jsonResults []health.CheckResultJSON

	for _, check := range checks {
		result := check()

		if doctorJSON {
			jsonResults = append(jsonResults, health.CheckResultJSON(result))
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

		if doctorQuiet && result.Status != "fail" {
			continue
		}

		icon := statusIcon(result.Status)
		fmt.Printf("%s %s\n", icon, fmtBold(result.Name))
		fmt.Printf("   %s\n", result.Message)

		if result.FixCmd != "" {
			fmt.Printf("   %sRecommendation: %s%s\n", colorCyan, result.FixCmd, colorReset)
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
			return fmt.Errorf("doctor found %d failed check(s)", failed)
		}
		return nil
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

	if failed > 0 {
		return fmt.Errorf("doctor found %d failed check(s)", failed)
	}
	return nil
}
