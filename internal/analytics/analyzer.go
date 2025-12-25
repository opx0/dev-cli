package analytics

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"dev-cli/internal/storage"
)

type CommandStats struct {
	Pattern       string   `json:"pattern"`
	RunCount      int      `json:"run_count"`
	FailCount     int      `json:"fail_count"`
	FailRate      float64  `json:"fail_rate"`
	AvgDurationMs int64    `json:"avg_duration_ms"`
	KnownFixes    []string `json:"known_fixes,omitempty"`
}

type Suggestion struct {
	Category    string  `json:"category"`
	Priority    string  `json:"priority"`
	Description string  `json:"description"`
	Fix         string  `json:"fix,omitempty"`
	Score       float64 `json:"score"`
}

type ErrorCluster struct {
	Fingerprint uint64   `json:"fingerprint"`
	Pattern     string   `json:"pattern"`
	Samples     []string `json:"samples"`
	Count       int      `json:"count"`
	Solutions   []string `json:"solutions,omitempty"`
}

type Analyzer struct {
	db *sql.DB
}

func NewAnalyzer(db *sql.DB) *Analyzer {
	return &Analyzer{db: db}
}

func (a *Analyzer) GetStats(cmd string) (*CommandStats, error) {
	baseCmd := extractBaseCommand(cmd)

	rows, err := a.db.Query(`
		SELECT command, exit_code, duration_ms 
		FROM history WHERE command LIKE ? 
		ORDER BY timestamp DESC LIMIT 100`, baseCmd+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &CommandStats{Pattern: baseCmd}
	var totalDuration int64

	for rows.Next() {
		var command string
		var exitCode int
		var durationMs int64
		if err := rows.Scan(&command, &exitCode, &durationMs); err != nil {
			continue
		}
		stats.RunCount++
		totalDuration += durationMs
		if exitCode != 0 {
			stats.FailCount++
		}
	}

	if stats.RunCount > 0 {
		stats.FailRate = float64(stats.FailCount) / float64(stats.RunCount)
		stats.AvgDurationMs = totalDuration / int64(stats.RunCount)
	}

	signature := storage.GenerateErrorSignature(baseCmd, 1, "")
	if solutions, err := storage.GetSolutionsForError(a.db, signature); err == nil {
		for _, sol := range solutions {
			stats.KnownFixes = append(stats.KnownFixes, sol.SolutionCommand)
		}
	}

	return stats, nil
}

func (a *Analyzer) GetSuggestions(cmd string) []Suggestion {
	var results []Suggestion

	stats, err := a.GetStats(cmd)
	if err != nil || stats.RunCount < 3 {
		return results
	}

	if stats.FailRate > 0.5 && stats.RunCount >= 5 {
		sug := Suggestion{
			Category:    "high_failure_rate",
			Priority:    "medium",
			Description: fmt.Sprintf("⚠ Fails %d%% of the time (%d/%d)", int(stats.FailRate*100), stats.FailCount, stats.RunCount),
			Score:       stats.FailRate,
		}
		if len(stats.KnownFixes) > 0 {
			sug.Fix = stats.KnownFixes[0]
			sug.Priority = "high"
		}
		results = append(results, sug)
	}

	if len(stats.KnownFixes) > 0 {
		results = append(results, Suggestion{
			Category:    "known_fix",
			Priority:    "high",
			Description: "💡 Known fix available from history",
			Fix:         stats.KnownFixes[0],
			Score:       0.8,
		})
	}

	if stats.AvgDurationMs > 30000 && stats.RunCount >= 3 {
		results = append(results, Suggestion{
			Category:    "slow_command",
			Priority:    "low",
			Description: fmt.Sprintf("⏱ Average runtime: %ds", stats.AvgDurationMs/1000),
			Score:       0.6,
		})
	}

	return results
}

func (a *Analyzer) GetFailurePatterns(limit int) ([]CommandStats, error) {
	rows, err := a.db.Query(`
		SELECT command, exit_code, duration_ms 
		FROM history WHERE exit_code != 0 
		ORDER BY timestamp DESC LIMIT ?`, limit*10)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patternMap := make(map[string]*CommandStats)

	for rows.Next() {
		var command string
		var exitCode, durationMs int
		if err := rows.Scan(&command, &exitCode, &durationMs); err != nil {
			continue
		}

		pattern := extractBaseCommand(command)
		if s, exists := patternMap[pattern]; exists {
			s.FailCount++
			s.RunCount++
		} else {
			patternMap[pattern] = &CommandStats{Pattern: pattern, FailCount: 1, RunCount: 1}
		}
	}

	var patterns []CommandStats
	for _, s := range patternMap {
		s.FailRate = float64(s.FailCount) / float64(s.RunCount)
		patterns = append(patterns, *s)
	}

	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].FailCount > patterns[j].FailCount
	})

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

func (a *Analyzer) ClusterErrors(limit int) ([]ErrorCluster, error) {
	rows, err := a.db.Query(`
		SELECT command, details FROM history 
		WHERE exit_code != 0 
		ORDER BY timestamp DESC LIMIT ?`, limit*5)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clusterMap := make(map[uint64]*ErrorCluster)

	for rows.Next() {
		var command, details string
		if err := rows.Scan(&command, &details); err != nil {
			continue
		}

		fingerprint := computeSimhash(details)
		normalized := normalizeErrorText(details)

		if cluster, exists := clusterMap[fingerprint]; exists {
			cluster.Count++
			if len(cluster.Samples) < 3 {
				cluster.Samples = append(cluster.Samples, truncate(normalized, 100))
			}
		} else {
			clusterMap[fingerprint] = &ErrorCluster{
				Fingerprint: fingerprint,
				Pattern:     extractBaseCommand(command),
				Samples:     []string{truncate(normalized, 100)},
				Count:       1,
			}
		}
	}

	for fp, cluster := range clusterMap {
		sig := fmt.Sprintf("%s:%d", cluster.Pattern, fp%10000)
		if solutions, err := storage.GetSolutionsForError(a.db, sig); err == nil {
			for _, sol := range solutions {
				cluster.Solutions = append(cluster.Solutions, sol.SolutionCommand)
			}
		}
	}

	var clusters []ErrorCluster
	for _, c := range clusterMap {
		clusters = append(clusters, *c)
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Count > clusters[j].Count
	})

	if len(clusters) > limit {
		clusters = clusters[:limit]
	}

	return clusters, nil
}

func (a *Analyzer) FindSimilarErrors(errorText string) ([]ErrorCluster, error) {
	targetHash := computeSimhash(errorText)

	clusters, err := a.ClusterErrors(50)
	if err != nil {
		return nil, err
	}

	var matches []ErrorCluster
	for _, c := range clusters {
		if hammingDistance(targetHash, c.Fingerprint) < 10 {
			matches = append(matches, c)
		}
	}

	return matches, nil
}

func extractBaseCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return cmd
	}
	return parts[0]
}

func normalizeErrorText(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.Join(strings.Fields(text), " ")
}

func computeSimhash(text string) uint64 {
	words := strings.Fields(normalizeErrorText(text))
	if len(words) == 0 {
		return 0
	}

	var vector [64]int
	for _, word := range words {
		h := fnv.New64a()
		h.Write([]byte(word))
		hash := h.Sum64()

		for i := 0; i < 64; i++ {
			if hash&(1<<i) != 0 {
				vector[i]++
			} else {
				vector[i]--
			}
		}
	}

	var fingerprint uint64
	for i := 0; i < 64; i++ {
		if vector[i] > 0 {
			fingerprint |= 1 << i
		}
	}

	return fingerprint
}

func hammingDistance(a, b uint64) int {
	xor := a ^ b
	count := 0
	for xor != 0 {
		count += int(xor & 1)
		xor >>= 1
	}
	return count
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
