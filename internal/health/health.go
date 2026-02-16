// Package health provides system health checks for dev-cli dependencies.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	FixFunc func() error
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
		CheckDockerCompose,
		CheckOllama,
		CheckOllamaModel,
		CheckGPU,
		CheckDevlogsDir,
		CheckNetwork,
	}
}

// AttemptFix tries to auto-fix a failed check.
func AttemptFix(result CheckResult) error {
	if result.FixFunc != nil {
		return result.FixFunc()
	}
	if result.FixCmd != "" {
		cmd := exec.Command("sh", "-c", result.FixCmd)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return fmt.Errorf("no fix available")
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
				Status:  "fail",
				Message: "Permission denied - user not in docker group",
				FixCmd:  "sudo usermod -aG docker $USER && newgrp docker",
			}
		}
		if strings.Contains(string(output), "Cannot connect") || strings.Contains(err.Error(), "executable file not found") {
			return CheckResult{
				Name:    "Docker",
				Status:  "fail",
				Message: "Docker daemon not running",
				FixCmd:  "sudo systemctl start docker",
			}
		}
		return CheckResult{
			Name:    "Docker",
			Status:  "fail",
			Message: fmt.Sprintf("Docker check failed: %v", err),
		}
	}

	versionCmd := exec.Command("docker", "--version")
	versionOutput, _ := versionCmd.Output()
	version := strings.TrimSpace(string(versionOutput))

	return CheckResult{
		Name:    "Docker",
		Status:  "ok",
		Message: version,
	}
}

// CheckDockerCompose verifies Docker Compose is installed.
func CheckDockerCompose() CheckResult {
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		cmd := exec.Command("docker", "compose", "version", "--short")
		output, _ := cmd.Output()
		return CheckResult{
			Name:    "Docker Compose",
			Status:  "ok",
			Message: "Plugin v" + strings.TrimSpace(string(output)),
		}
	}

	if _, err := exec.LookPath("docker-compose"); err == nil {
		cmd := exec.Command("docker-compose", "--version")
		output, _ := cmd.Output()
		return CheckResult{
			Name:    "Docker Compose",
			Status:  "ok",
			Message: strings.TrimSpace(string(output)),
		}
	}

	return CheckResult{
		Name:    "Docker Compose",
		Status:  "fail",
		Message: "Docker Compose not installed",
		FixCmd:  "sudo pacman -S docker-compose",
		FixFunc: func() error {
			cmd := exec.Command("sudo", "pacman", "-S", "--noconfirm", "docker-compose")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}

// CheckOllama verifies Ollama is running and accessible.
func CheckOllama() CheckResult {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")

	if err == nil && resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			return CheckResult{
				Name:    "Ollama",
				Status:  "ok",
				Message: "Running on localhost:11434",
			}
		}
	}

	dockerCheck := exec.Command("docker", "ps", "-a", "--filter", "name=ollama", "--format", "{{.Names}}")
	output, _ := dockerCheck.Output()
	containerExists := strings.TrimSpace(string(output)) != ""

	if containerExists {
		return CheckResult{
			Name:    "Ollama",
			Status:  "fail",
			Message: "Ollama container exists but not responding",
			FixCmd:  "docker start ollama",
			FixFunc: func() error {
				cmd := exec.Command("docker", "start", "ollama")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			},
		}
	}

	projectRoot := findProjectRoot()
	composeFile := filepath.Join(projectRoot, "infra", "ollama", "docker-compose.yml")
	if _, err := os.Stat(composeFile); err == nil {
		return CheckResult{
			Name:    "Ollama",
			Status:  "fail",
			Message: "Ollama not running (Docker compose available)",
			FixCmd:  "cd infra/ollama && docker compose up -d",
			FixFunc: func() error {
				return runDockerCompose(composeFile, "up", "-d")
			},
		}
	}

	if _, err := exec.LookPath("ollama"); err == nil {
		return CheckResult{
			Name:    "Ollama",
			Status:  "fail",
			Message: "Ollama installed but not running",
			FixCmd:  "ollama serve &",
		}
	}

	return CheckResult{
		Name:    "Ollama",
		Status:  "fail",
		Message: "Ollama not installed",
		FixCmd:  "cd infra/ollama && docker compose up -d",
		FixFunc: func() error {
			if projectRoot != "" {
				cf := filepath.Join(projectRoot, "infra", "ollama", "docker-compose.yml")
				if _, err := os.Stat(cf); err == nil {
					return runDockerCompose(cf, "up", "-d")
				}
			}
			return fmt.Errorf("docker-compose.yml not found - run from project root")
		},
	}
}

// CheckOllamaModel checks whether at least one model is installed in Ollama.
func CheckOllamaModel() CheckResult {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")

	if err != nil {
		return CheckResult{
			Name:    "Ollama Model",
			Status:  "warn",
			Message: "Cannot check models - Ollama not running",
		}
	}
	defer resp.Body.Close()

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
			FixCmd:  "ollama pull llama3.2",
			FixFunc: func() error {
				cmd := exec.Command("ollama", "pull", "llama3.2")
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			},
		}
	}

	return CheckResult{
		Name:    "Ollama Model",
		Status:  "ok",
		Message: fmt.Sprintf("Model(s) available (%s)", tagsResp.Models[0].Name),
	}
}

// CheckGPU checks for NVIDIA GPU availability.
func CheckGPU() CheckResult {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader")
	output, err := cmd.Output()

	if err != nil {
		return CheckResult{
			Name:    "GPU (NVIDIA)",
			Status:  "warn",
			Message: "NVIDIA GPU not detected (CPU mode will be used)",
		}
	}

	gpuInfo := strings.TrimSpace(string(output))
	return CheckResult{
		Name:    "GPU (NVIDIA)",
		Status:  "ok",
		Message: gpuInfo,
	}
}

// CheckDevlogsDir verifies the ~/.devlogs directory exists.
func CheckDevlogsDir() CheckResult {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{
			Name:    "Devlogs Directory",
			Status:  "fail",
			Message: "Cannot determine home directory",
		}
	}

	devlogsDir := filepath.Join(homeDir, ".devlogs")

	if _, err := os.Stat(devlogsDir); os.IsNotExist(err) {
		return CheckResult{
			Name:    "Devlogs Directory",
			Status:  "warn",
			Message: fmt.Sprintf("%s does not exist", devlogsDir),
			FixFunc: func() error {
				return os.MkdirAll(devlogsDir, 0755)
			},
		}
	}

	return CheckResult{
		Name:    "Devlogs Directory",
		Status:  "ok",
		Message: devlogsDir,
	}
}

// CheckNetwork verifies external API connectivity.
func CheckNetwork() CheckResult {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.perplexity.ai")

	if err != nil {
		return CheckResult{
			Name:    "Network",
			Status:  "warn",
			Message: "Cannot reach external APIs (cloud AI features may not work)",
		}
	}
	defer resp.Body.Close()

	return CheckResult{
		Name:    "Network",
		Status:  "ok",
		Message: "External APIs reachable",
	}
}

// ── Internal helpers ─────────────────────────────────────────────────────────

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func getDockerComposeCmd() (string, []string) {
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		return "docker", []string{"compose"}
	}
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return "docker-compose", []string{}
	}
	return "docker", []string{"compose"}
}

func runDockerCompose(composeFile string, args ...string) error {
	bin, prefix := getDockerComposeCmd()
	cmdArgs := append(prefix, "-f", composeFile)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
