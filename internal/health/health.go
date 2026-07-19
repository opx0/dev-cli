// Package health provides system health checks for dev-cli dependencies.
package health

import (
	"context"
	"dev-cli/internal/config"
	"dev-cli/internal/llm"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ── Types ────────────────────────────────────────────────────────────────────

// CheckResult holds the outcome of a single health check.
type CheckResult struct {
	Name    string
	Status  string // "ok", "warn", "fail"
	Message string
	FixCmd  string
}

// Report is the JSON output format for agent/CI consumption.
type Report struct {
	Timestamp string            `json:"timestamp"`
	Checks    []CheckResultJSON `json:"checks"`
	Summary   Summary           `json:"summary"`
}

// CheckResultJSON is the JSON-serializable version of CheckResult.
type CheckResultJSON struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
	FixCmd  string `json:"fix_cmd,omitempty"`
}

// Summary contains aggregate check results.
type Summary struct {
	Passed int `json:"passed"`
	Warned int `json:"warned"`
	Failed int `json:"failed"`
	Total  int `json:"total"`
}

// ── Run all checks ──────────────────────────────────────────────────────────

// AllChecks returns the standard set of health check functions.
func AllChecks() []func() CheckResult {
	return []func() CheckResult{
		CheckDocker,
		CheckOllama,
		CheckOllamaModel,
		CheckLLMProvider,
		CheckDevlogsDir,
	}
}

// CheckLLMProvider verifies that at least one LLM provider is configured and
// reachable — either Ollama (local) or an OpenAI-compatible cloud endpoint.
// This is the check that tells a user "you can actually run `dev-cli fix`".
func CheckLLMProvider() CheckResult {
	cfg := config.Load()

	ollamaURL := strings.TrimSuffix(cfg.OllamaURL, "/")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if resp, err := get(ctx, client, ollamaURL+"/api/tags"); err == nil && resp != nil {
		_ = resp.Body.Close()
		if resp.StatusCode == 200 {
			return CheckResult{
				Name:    "LLM Provider",
				Status:  "ok",
				Message: fmt.Sprintf("Ollama reachable at %s", ollamaURL),
			}
		}
	}

	if cfg.OpenAIKey != "" {
		provider := llm.NewOpenAIProvider(cfg)
		if provider != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := provider.Ping(ctx); err == nil {
				return CheckResult{
					Name:    "LLM Provider",
					Status:  "ok",
					Message: fmt.Sprintf("OpenAI-compatible endpoint reachable (%s)", cfg.OpenAIModel),
				}
			} else {
				return CheckResult{
					Name:    "LLM Provider",
					Status:  "fail",
					Message: fmt.Sprintf("OPENAI_API_KEY set but endpoint unreachable: %v", err),
				}
			}
		}
	}

	return CheckResult{
		Name:    "LLM Provider",
		Status:  "fail",
		Message: "No LLM provider available (no Ollama running, no OPENAI_API_KEY set)",
		FixCmd:  "Start Ollama or configure OPENAI_API_KEY",
	}
}

// ── Individual checks ────────────────────────────────────────────────────────

// CheckDocker verifies the Docker daemon is running and accessible.
func CheckDocker() CheckResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info")
	output, err := cmd.CombinedOutput()

	if err != nil {
		if strings.Contains(string(output), "permission denied") {
			return CheckResult{
				Name:    "Docker",
				Status:  "warn",
				Message: "Permission denied - user not in docker group",
				FixCmd:  "sudo usermod -aG docker $USER && newgrp docker",
			}
		}
		if strings.Contains(string(output), "Cannot connect") || strings.Contains(err.Error(), "executable file not found") {
			return CheckResult{
				Name:    "Docker",
				Status:  "warn",
				Message: "Docker daemon not running",
				FixCmd:  "sudo systemctl start docker",
			}
		}
		return CheckResult{
			Name:    "Docker",
			Status:  "warn",
			Message: fmt.Sprintf("Docker check failed: %v", err),
		}
	}

	versionCmd := exec.CommandContext(ctx, "docker", "--version")
	versionOutput, _ := versionCmd.Output()
	version := strings.TrimSpace(string(versionOutput))

	return CheckResult{
		Name:    "Docker",
		Status:  "ok",
		Message: version,
	}
}

// CheckOllama verifies Ollama is running and accessible.
func CheckOllama() CheckResult {
	cfg := config.Load()
	baseURL := strings.TrimSuffix(cfg.OllamaURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := get(ctx, client, baseURL+"/api/tags")

	if err == nil && resp != nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode == 200 {
			return CheckResult{
				Name:    "Ollama",
				Status:  "ok",
				Message: fmt.Sprintf("Running on %s", baseURL),
			}
		}
	}

	dockerCtx, dockerCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer dockerCancel()
	dockerCheck := exec.CommandContext(dockerCtx, "docker", "ps", "-a", "--filter", "name=ollama", "--format", "{{.Names}}")
	output, _ := dockerCheck.Output()
	containerExists := strings.TrimSpace(string(output)) != ""

	if containerExists {
		return CheckResult{
			Name:    "Ollama",
			Status:  "warn",
			Message: "Ollama container exists but not responding",
			FixCmd:  "docker start ollama",
		}
	}

	if _, err := exec.LookPath("ollama"); err == nil {
		return CheckResult{
			Name:    "Ollama",
			Status:  "warn",
			Message: "Ollama installed but not running",
			FixCmd:  "ollama serve &",
		}
	}

	return CheckResult{
		Name:    "Ollama",
		Status:  "warn",
		Message: "Ollama not installed",
		FixCmd:  "Install Ollama, then run: ollama serve",
	}
}

// CheckOllamaModel checks whether at least one model is installed in Ollama.
func CheckOllamaModel() CheckResult {
	cfg := config.Load()
	baseURL := strings.TrimSuffix(cfg.OllamaURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	targetModel := cfg.OllamaModel
	if targetModel == "" {
		targetModel = "smallthinker"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := get(ctx, client, baseURL+"/api/tags")

	if err != nil {
		return CheckResult{
			Name:    "Ollama Model",
			Status:  "warn",
			Message: fmt.Sprintf("Cannot check models - Ollama not running at %s", baseURL),
		}
	}
	defer func() { _ = resp.Body.Close() }()

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil || len(tagsResp.Models) == 0 {
		return CheckResult{
			Name:    "Ollama Model",
			Status:  "warn",
			Message: "No models installed",
			FixCmd:  fmt.Sprintf("ollama pull %s", targetModel),
		}
	}

	target := strings.ToLower(targetModel)
	hasTarget := false
	available := make([]string, 0, len(tagsResp.Models))
	for _, model := range tagsResp.Models {
		available = append(available, model.Name)
		name := strings.ToLower(model.Name)
		if name == target || strings.HasPrefix(name, target+":") {
			hasTarget = true
		}
	}

	if !hasTarget {
		return CheckResult{
			Name:    "Ollama Model",
			Status:  "warn",
			Message: fmt.Sprintf("Configured model '%s' not found (available: %s)", targetModel, strings.Join(available, ", ")),
			FixCmd:  fmt.Sprintf("ollama pull %s", targetModel),
		}
	}

	return CheckResult{
		Name:    "Ollama Model",
		Status:  "ok",
		Message: fmt.Sprintf("Configured model ready (%s)", targetModel),
	}
}

func get(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

// CheckDevlogsDir verifies the ~/.devlogs directory exists.
func CheckDevlogsDir() CheckResult {
	devlogsDir := config.Load().LogDir

	if _, err := os.Stat(devlogsDir); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Devlogs Directory",
			Status:  "warn",
			Message: fmt.Sprintf("%s does not exist", devlogsDir),
		}
	}

	return CheckResult{
		Name:    "Devlogs Directory",
		Status:  "ok",
		Message: devlogsDir,
	}
}
