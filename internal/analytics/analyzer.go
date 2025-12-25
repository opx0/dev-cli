// Package analytics provides proactive debugging insights from command history.
package analytics

import (
	"database/sql"
	"sort"
	"strings"

	"dev-cli/internal/storage"
)

// CommandStats represents failure statistics for a command pattern.
type CommandStats struct {
	CommandPattern string   `json:"command_pattern"`
	TotalRuns      int      `json:"total_runs"`
	FailureCount   int      `json:"failure_count"`
	FailureRate    float64  `json:"failure_rate"` // 0.0 - 1.0
	AvgDurationMs  int64    `json:"avg_duration_ms"`
	CommonFixes    []string `json:"common_fixes,omitempty"`
	LastFailure    string   `json:"last_failure,omitempty"`
}

// ProactiveSuggestion represents a debugging suggestion based on history patterns.
type ProactiveSuggestion struct {
	Type         string  `json:"type"`     // "high_failure_rate", "known_fix", "slow_command"
	Severity     string  `json:"severity"` // "high", "medium", "low"
	Message      string  `json:"message"`
	SuggestedFix string  `json:"suggested_fix,omitempty"`
	Confidence   float64 `json:"confidence"` // 0.0 - 1.0
}

// Analyzer provides proactive debugging analysis capabilities.
type Analyzer struct {
	db *sql.DB
}

// NewAnalyzer creates a new analytics analyzer.
func NewAnalyzer(db *sql.DB) *Analyzer {
	return &Analyzer{db: db}
}

// GetCommandStats returns failure statistics for a command pattern.
func (a *Analyzer) GetCommandStats(commandPattern string) (*CommandStats, error) {
	// Normalize command pattern (first word/binary)
	pattern := normalizeCommand(commandPattern)

	query := `SELECT command, exit_code, duration_ms 
			  FROM history 
			  WHERE command LIKE ? 
			  ORDER BY timestamp DESC 
			  LIMIT 100`

	rows, err := a.db.Query(query, pattern+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &CommandStats{
		CommandPattern: pattern,
	}

	var totalDuration int64

	for rows.Next() {
		var command string
		var exitCode int
		var durationMs int64
		if err := rows.Scan(&command, &exitCode, &durationMs); err != nil {
			continue
		}
		stats.TotalRuns++
		totalDuration += durationMs
		if exitCode != 0 {
			stats.FailureCount++
		}
	}

	if stats.TotalRuns > 0 {
		stats.FailureRate = float64(stats.FailureCount) / float64(stats.TotalRuns)
		stats.AvgDurationMs = totalDuration / int64(stats.TotalRuns)
	}

	// Look for known solutions
	signature := storage.GenerateErrorSignature(pattern, 1, "")
	if solutions, err := storage.GetSolutionsForError(a.db, signature); err == nil && len(solutions) > 0 {
		for _, sol := range solutions {
			stats.CommonFixes = append(stats.CommonFixes, sol.SolutionCommand)
		}
	}

	return stats, nil
}

// GetProactiveSuggestions analyzes a command and returns debugging suggestions.
func (a *Analyzer) GetProactiveSuggestions(command string) []ProactiveSuggestion {
	var suggestions []ProactiveSuggestion

	stats, err := a.GetCommandStats(command)
	if err != nil || stats.TotalRuns < 3 {
		return suggestions // Not enough data
	}

	// High failure rate suggestion
	if stats.FailureRate > 0.5 && stats.TotalRuns >= 5 {
		sug := ProactiveSuggestion{
			Type:       "high_failure_rate",
			Severity:   "medium",
			Message:    formatFailureRateMessage(stats),
			Confidence: stats.FailureRate,
		}
		if len(stats.CommonFixes) > 0 {
			sug.SuggestedFix = stats.CommonFixes[0]
			sug.Severity = "high"
		}
		suggestions = append(suggestions, sug)
	}

	// Known fix suggestion
	if len(stats.CommonFixes) > 0 {
		suggestions = append(suggestions, ProactiveSuggestion{
			Type:         "known_fix",
			Severity:     "high",
			Message:      "A fix for similar errors is known from your history",
			SuggestedFix: stats.CommonFixes[0],
			Confidence:   0.8,
		})
	}

	// Slow command warning
	if stats.AvgDurationMs > 30000 && stats.TotalRuns >= 3 {
		suggestions = append(suggestions, ProactiveSuggestion{
			Type:       "slow_command",
			Severity:   "low",
			Message:    formatSlowCommandMessage(stats),
			Confidence: 0.6,
		})
	}

	return suggestions
}

// GetRecentFailurePatterns identifies recurring failures in recent history.
func (a *Analyzer) GetRecentFailurePatterns(limit int) ([]CommandStats, error) {
	query := `SELECT command, exit_code, duration_ms 
			  FROM history 
			  WHERE exit_code != 0 
			  ORDER BY timestamp DESC 
			  LIMIT ?`

	rows, err := a.db.Query(query, limit*10) // Get more to aggregate
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Aggregate by command pattern
	patternMap := make(map[string]*CommandStats)

	for rows.Next() {
		var command string
		var exitCode int
		var durationMs int64
		if err := rows.Scan(&command, &exitCode, &durationMs); err != nil {
			continue
		}

		pattern := normalizeCommand(command)
		if stats, ok := patternMap[pattern]; ok {
			stats.FailureCount++
			stats.TotalRuns++
		} else {
			patternMap[pattern] = &CommandStats{
				CommandPattern: pattern,
				FailureCount:   1,
				TotalRuns:      1,
			}
		}
	}

	// Convert to slice and sort by failure count
	var patterns []CommandStats
	for _, stats := range patternMap {
		stats.FailureRate = float64(stats.FailureCount) / float64(stats.TotalRuns)
		patterns = append(patterns, *stats)
	}

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].FailureCount > patterns[j].FailureCount
	})

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

// normalizeCommand extracts the base command from a full command line.
func normalizeCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return command
	}
	return parts[0]
}

func formatFailureRateMessage(stats *CommandStats) string {
	pct := int(stats.FailureRate * 100)
	return strings.Replace(
		"⚠ This command fails RATE% of the time (COUNT/TOTAL runs)",
		"RATE", string(rune('0'+pct/10))+string(rune('0'+pct%10)),
		1,
	)
}

func formatSlowCommandMessage(stats *CommandStats) string {
	secs := stats.AvgDurationMs / 1000
	return strings.Replace(
		"⏱ This command typically takes SECSs to complete",
		"SECS", string(rune('0'+secs/10))+string(rune('0'+secs%10)),
		1,
	)
}
