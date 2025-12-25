package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dev-cli/internal/storage"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerQueryHistoryTool adds the query_command_history tool to the server.
func registerQueryHistoryTool(s *server.MCPServer) {
	tool := mcp.NewTool("query_command_history",
		mcp.WithDescription("Search past shell commands by pattern, exit code, or time range. Returns command history with execution details."),
		mcp.WithString("pattern",
			mcp.Description("Search pattern to match commands (e.g., 'npm', 'docker', 'git')"),
		),
		mcp.WithNumber("exit_code",
			mcp.Description("Filter by specific exit code (e.g., 1 for failures, 0 for success)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 20)"),
		),
		mcp.WithString("since",
			mcp.Description("Time range like '1h', '30m', '7d' to filter recent commands"),
		),
	)

	s.AddTool(tool, queryHistoryHandler)
}

func queryHistoryHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	// Parse arguments
	pattern := ""
	if p, ok := args["pattern"].(string); ok {
		pattern = p
	}

	limit := 20
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var sinceDur time.Duration
	if s, ok := args["since"].(string); ok && s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			sinceDur = d
		}
	}

	// Query database
	db, err := storage.InitDB()
	if err != nil {
		return mcp.NewToolResultError("failed to open database: " + err.Error()), nil
	}
	defer db.Close()

	opts := storage.QueryOpts{
		Limit:  limit,
		Filter: pattern,
		Since:  sinceDur,
	}

	var items []storage.HistoryItem

	// If checking for failures
	if exitCode, ok := args["exit_code"].(float64); ok && exitCode != 0 {
		items, err = storage.GetFailures(db, opts)
	} else if pattern != "" {
		items, err = storage.SearchHistory(db, pattern)
		if len(items) > limit {
			items = items[:limit]
		}
	} else {
		items, err = storage.GetRecentHistory(db, limit)
	}

	if err != nil {
		return mcp.NewToolResultError("query failed: " + err.Error()), nil
	}

	result, _ := json.MarshalIndent(items, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// registerFindSimilarFailuresTool adds the find_similar_failures tool to the server.
func registerFindSimilarFailuresTool(s *server.MCPServer) {
	tool := mcp.NewTool("find_similar_failures",
		mcp.WithDescription("Find past command failures with similar error patterns. Useful for identifying recurring issues."),
		mcp.WithString("error_text",
			mcp.Required(),
			mcp.Description("The error message or output to match against"),
		),
		mcp.WithString("command",
			mcp.Description("The command that failed (for generating signature)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum similar failures to return (default: 5)"),
		),
	)

	s.AddTool(tool, findSimilarHandler)
}

func findSimilarHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	errorText := ""
	if e, ok := args["error_text"].(string); ok {
		errorText = e
	}

	command := ""
	if c, ok := args["command"].(string); ok {
		command = c
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	// Generate error signature
	signature := storage.GenerateErrorSignature(command, 1, errorText)

	db, err := storage.InitDB()
	if err != nil {
		return mcp.NewToolResultError("failed to open database: " + err.Error()), nil
	}
	defer db.Close()

	items, err := storage.GetSimilarFailures(db, signature, limit)
	if err != nil {
		return mcp.NewToolResultError("query failed: " + err.Error()), nil
	}

	result := map[string]interface{}{
		"error_signature":  signature,
		"similar_failures": items,
		"count":            len(items),
	}
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// registerGetSolutionsTool adds the get_known_solutions tool to the server.
func registerGetSolutionsTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_known_solutions",
		mcp.WithDescription("Get previously successful solutions for an error pattern. Returns commands that fixed similar issues."),
		mcp.WithString("error_text",
			mcp.Required(),
			mcp.Description("The error message to find solutions for"),
		),
		mcp.WithString("command",
			mcp.Description("The command that produced the error"),
		),
	)

	s.AddTool(tool, getSolutionsHandler)
}

func getSolutionsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	errorText := ""
	if e, ok := args["error_text"].(string); ok {
		errorText = e
	}

	command := ""
	if c, ok := args["command"].(string); ok {
		command = c
	}

	signature := storage.GenerateErrorSignature(command, 1, errorText)

	db, err := storage.InitDB()
	if err != nil {
		return mcp.NewToolResultError("failed to open database: " + err.Error()), nil
	}
	defer db.Close()

	solutions, err := storage.GetSolutionsForError(db, signature)
	if err != nil {
		return mcp.NewToolResultError("query failed: " + err.Error()), nil
	}

	result := map[string]interface{}{
		"error_signature": signature,
		"solutions":       solutions,
		"count":           len(solutions),
	}
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// registerStoreSolutionTool adds the store_solution tool to the server.
func registerStoreSolutionTool(s *server.MCPServer) {
	tool := mcp.NewTool("store_solution",
		mcp.WithDescription("Store a successful solution for an error pattern. This teaches dev-cli to suggest this fix for similar future errors."),
		mcp.WithString("error_text",
			mcp.Required(),
			mcp.Description("The error message that was fixed"),
		),
		mcp.WithString("solution_command",
			mcp.Required(),
			mcp.Description("The command that fixed the error"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description of why this solution works"),
		),
	)

	s.AddTool(tool, storeSolutionHandler)
}

func storeSolutionHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	errorText := ""
	if e, ok := args["error_text"].(string); ok {
		errorText = e
	}

	solutionCmd := ""
	if s, ok := args["solution_command"].(string); ok {
		solutionCmd = s
	}

	description := ""
	if d, ok := args["description"].(string); ok {
		description = d
	}

	if errorText == "" || solutionCmd == "" {
		return mcp.NewToolResultError("error_text and solution_command are required"), nil
	}

	signature := storage.GenerateErrorSignature("", 1, errorText)

	db, err := storage.InitDB()
	if err != nil {
		return mcp.NewToolResultError("failed to open database: " + err.Error()), nil
	}
	defer db.Close()

	err = storage.StoreSolution(db, signature, solutionCmd, description)
	if err != nil {
		return mcp.NewToolResultError("failed to store solution: " + err.Error()), nil
	}

	result := map[string]interface{}{
		"status":          "stored",
		"error_signature": signature,
		"solution":        solutionCmd,
	}
	jsonResult, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// registerGetProjectFingerprintTool adds the get_project_fingerprint tool.
func registerGetProjectFingerprintTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_project_fingerprint",
		mcp.WithDescription("Detect the project type and common issues for a directory. Returns package manager, project type, and associated runbooks."),
		mcp.WithString("path",
			mcp.Description("Path to analyze (default: current directory)"),
		),
	)

	s.AddTool(tool, getProjectFingerprintHandler)
}

func getProjectFingerprintHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	path := "."
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	// Simple project detection based on files
	fingerprint := detectProjectType(path)

	jsonResult, _ := json.MarshalIndent(fingerprint, "", "  ")
	return mcp.NewToolResultText(string(jsonResult)), nil
}

// detectProjectType analyzes a directory to determine project characteristics.
func detectProjectType(path string) map[string]interface{} {
	result := map[string]interface{}{
		"path":            path,
		"project_type":    "unknown",
		"package_manager": "unknown",
		"common_issues":   []string{},
		"detected_files":  []string{},
	}

	// Project indicators: filename → (type, manager, issues)
	type projectInfo struct {
		projectType    string
		packageManager string
		commonIssues   []string
	}

	indicators := map[string]projectInfo{
		"package.json":        {"nodejs", "npm", []string{"EACCES permission denied", "node_modules corruption", "version conflicts"}},
		"package-lock.json":   {"nodejs", "npm", []string{"lockfile conflicts", "integrity check failures"}},
		"yarn.lock":           {"nodejs", "yarn", []string{"resolution failures", "cache issues"}},
		"pnpm-lock.yaml":      {"nodejs", "pnpm", []string{"store issues", "peer dependency conflicts"}},
		"go.mod":              {"go", "go mod", []string{"module not found", "version mismatch", "build cache issues"}},
		"go.sum":              {"go", "go mod", []string{"checksum mismatch", "module verification"}},
		"requirements.txt":    {"python", "pip", []string{"module not found", "version conflicts", "virtualenv issues"}},
		"Pipfile":             {"python", "pipenv", []string{"lock failures", "python version mismatch"}},
		"pyproject.toml":      {"python", "poetry/pip", []string{"build failures", "dependency resolution"}},
		"Cargo.toml":          {"rust", "cargo", []string{"dependency resolution", "build failures", "linking errors"}},
		"docker-compose.yml":  {"docker", "docker-compose", []string{"port conflicts", "network issues", "volume permissions"}},
		"docker-compose.yaml": {"docker", "docker-compose", []string{"port conflicts", "network issues", "volume permissions"}},
		"Dockerfile":          {"docker", "docker", []string{"build failures", "layer caching", "base image issues"}},
		"Makefile":            {"make", "make", []string{"missing targets", "shell errors", "dependency issues"}},
		".gitlab-ci.yml":      {"ci/cd", "gitlab-ci", []string{"pipeline failures", "runner issues", "artifact problems"}},
		".github":             {"ci/cd", "github-actions", []string{"workflow failures", "secret issues", "runner problems"}},
		"Gemfile":             {"ruby", "bundler", []string{"gem conflicts", "native extension failures"}},
		"composer.json":       {"php", "composer", []string{"autoload issues", "memory limits", "version conflicts"}},
	}

	detectedFiles := []string{}
	allIssues := []string{}

	// Check each indicator file
	for filename, info := range indicators {
		fullPath := filepath.Join(path, filename)
		if _, err := os.Stat(fullPath); err == nil {
			detectedFiles = append(detectedFiles, filename)
			// First match sets primary type
			if result["project_type"] == "unknown" {
				result["project_type"] = info.projectType
				result["package_manager"] = info.packageManager
			}
			// Aggregate all common issues
			allIssues = append(allIssues, info.commonIssues...)
		}
	}

	// Remove duplicate issues
	uniqueIssues := removeDuplicates(allIssues)
	result["common_issues"] = uniqueIssues
	result["detected_files"] = detectedFiles

	// Add project-specific context
	if len(detectedFiles) == 0 {
		result["note"] = "No recognizable project files found"
	} else {
		result["note"] = fmt.Sprintf("Detected %d project files", len(detectedFiles))
	}

	return result
}

// removeDuplicates returns unique strings from a slice.
func removeDuplicates(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// registerGetSuggestionsTool registers the get_proactive_suggestions tool.
func registerGetSuggestionsTool(s *server.MCPServer) {
	tool := mcp.NewTool("get_proactive_suggestions",
		mcp.WithDescription("Get proactive debugging suggestions based on command history patterns. Returns failure rates, known fixes, and warnings for commands that frequently fail."),
		mcp.WithString("command",
			mcp.Description("Command to analyze for suggestions (optional - if empty, returns recent failure patterns)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of suggestions to return (default: 5)"),
		),
	)

	s.AddTool(tool, getSuggestionsHandler)
}

func getSuggestionsHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	command, _ := args["command"].(string)
	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	db, err := storage.InitDB()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to init DB: %v", err)), nil
	}
	defer db.Close()

	result := make(map[string]interface{})

	if command != "" {
		// Get suggestions for specific command
		stats, err := getCommandStatsFromDB(db, command)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get stats: %v", err)), nil
		}

		suggestions := generateSuggestions(stats)
		result["command"] = command
		result["stats"] = stats
		result["suggestions"] = suggestions
	} else {
		// Get recent failure patterns
		patterns, err := getRecentFailurePatternsFromDB(db, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get patterns: %v", err)), nil
		}
		result["failure_patterns"] = patterns
		result["count"] = len(patterns)
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

type commandStats struct {
	CommandPattern string   `json:"command_pattern"`
	TotalRuns      int      `json:"total_runs"`
	FailureCount   int      `json:"failure_count"`
	FailureRate    float64  `json:"failure_rate"`
	AvgDurationMs  int64    `json:"avg_duration_ms"`
	CommonFixes    []string `json:"common_fixes,omitempty"`
}

type proactiveSuggestion struct {
	Type         string  `json:"type"`
	Severity     string  `json:"severity"`
	Message      string  `json:"message"`
	SuggestedFix string  `json:"suggested_fix,omitempty"`
	Confidence   float64 `json:"confidence"`
}

func getCommandStatsFromDB(db *sql.DB, command string) (*commandStats, error) {
	// Get base command
	parts := strings.Fields(command)
	pattern := command
	if len(parts) > 0 {
		pattern = parts[0]
	}

	query := `SELECT command, exit_code, duration_ms 
			  FROM history 
			  WHERE command LIKE ? 
			  ORDER BY timestamp DESC 
			  LIMIT 100`

	rows, err := db.Query(query, pattern+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &commandStats{
		CommandPattern: pattern,
	}
	var totalDuration int64

	for rows.Next() {
		var cmd string
		var exitCode int
		var durationMs int64
		if err := rows.Scan(&cmd, &exitCode, &durationMs); err != nil {
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
	if solutions, err := storage.GetSolutionsForError(db, signature); err == nil && len(solutions) > 0 {
		for _, sol := range solutions {
			stats.CommonFixes = append(stats.CommonFixes, sol.SolutionCommand)
		}
	}

	return stats, nil
}

func generateSuggestions(stats *commandStats) []proactiveSuggestion {
	var suggestions []proactiveSuggestion

	if stats.TotalRuns < 3 {
		return suggestions
	}

	// High failure rate
	if stats.FailureRate > 0.5 && stats.TotalRuns >= 5 {
		pct := int(stats.FailureRate * 100)
		sug := proactiveSuggestion{
			Type:       "high_failure_rate",
			Severity:   "medium",
			Message:    fmt.Sprintf("⚠ This command fails %d%% of the time (%d/%d runs)", pct, stats.FailureCount, stats.TotalRuns),
			Confidence: stats.FailureRate,
		}
		if len(stats.CommonFixes) > 0 {
			sug.SuggestedFix = stats.CommonFixes[0]
			sug.Severity = "high"
		}
		suggestions = append(suggestions, sug)
	}

	// Known fix
	if len(stats.CommonFixes) > 0 {
		suggestions = append(suggestions, proactiveSuggestion{
			Type:         "known_fix",
			Severity:     "high",
			Message:      "💡 A fix for similar errors is known from your history",
			SuggestedFix: stats.CommonFixes[0],
			Confidence:   0.8,
		})
	}

	// Slow command
	if stats.AvgDurationMs > 30000 && stats.TotalRuns >= 3 {
		secs := stats.AvgDurationMs / 1000
		suggestions = append(suggestions, proactiveSuggestion{
			Type:       "slow_command",
			Severity:   "low",
			Message:    fmt.Sprintf("⏱ This command typically takes %ds to complete", secs),
			Confidence: 0.6,
		})
	}

	return suggestions
}

func getRecentFailurePatternsFromDB(db *sql.DB, limit int) ([]commandStats, error) {
	query := `SELECT command, exit_code, duration_ms 
			  FROM history 
			  WHERE exit_code != 0 
			  ORDER BY timestamp DESC 
			  LIMIT ?`

	rows, err := db.Query(query, limit*10)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patternMap := make(map[string]*commandStats)

	for rows.Next() {
		var command string
		var exitCode int
		var durationMs int64
		if err := rows.Scan(&command, &exitCode, &durationMs); err != nil {
			continue
		}

		parts := strings.Fields(command)
		pattern := command
		if len(parts) > 0 {
			pattern = parts[0]
		}

		if stats, ok := patternMap[pattern]; ok {
			stats.FailureCount++
			stats.TotalRuns++
		} else {
			patternMap[pattern] = &commandStats{
				CommandPattern: pattern,
				FailureCount:   1,
				TotalRuns:      1,
			}
		}
	}

	var patterns []commandStats
	for _, stats := range patternMap {
		stats.FailureRate = float64(stats.FailureCount) / float64(stats.TotalRuns)
		patterns = append(patterns, *stats)
	}

	// Sort by failure count descending
	for i := 0; i < len(patterns)-1; i++ {
		for j := i + 1; j < len(patterns); j++ {
			if patterns[j].FailureCount > patterns[i].FailureCount {
				patterns[i], patterns[j] = patterns[j], patterns[i]
			}
		}
	}

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns, nil
}
