package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
