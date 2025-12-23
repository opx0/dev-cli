package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	doctorFix   bool
	doctorQuiet bool
)

type CheckResult struct {
	Name    string
	Status  string // "ok", "warn", "fail"
	Message string
	FixCmd  string
	FixFunc func() error
}

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
}

func runDoctor(cmd *cobra.Command, args []string) {
	fmt.Println("\033[1m🔍 dev-cli doctor\033[0m")
	fmt.Println()

	checks := []func() CheckResult{
		checkDocker,
		checkDockerCompose,
		checkOllama,
		checkOllamaModel,
		checkGPU,
		checkDevlogsDir,
		checkNetwork,
	}

	var failed, warned, passed int

	for _, check := range checks {
		result := check()

		if doctorQuiet && result.Status == "ok" {
			passed++
			continue
		}

		icon := getStatusIcon(result.Status)
		fmt.Printf("%s \033[1m%s\033[0m\n", icon, result.Name)
		fmt.Printf("   %s\n", result.Message)

		switch result.Status {
		case "ok":
			passed++
		case "warn":
			warned++
		case "fail":
			failed++
			if doctorFix && (result.FixCmd != "" || result.FixFunc != nil) {
				fmt.Printf("   \033[33m➜ Attempting fix...\033[0m\n")
				if err := attemptFix(result); err != nil {
					fmt.Printf("   \033[31m✗ Fix failed: %v\033[0m\n", err)
				} else {
					fmt.Printf("   \033[32m✓ Fixed!\033[0m\n")
					failed--
					passed++
				}
			} else if result.FixCmd != "" {
				fmt.Printf("   \033[36m💡 Fix: %s\033[0m\n", result.FixCmd)
			}
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("\033[90m────────────────────────────────\033[0m")
	fmt.Printf("✓ %d passed  ", passed)
	if warned > 0 {
		fmt.Printf("⚠ %d warnings  ", warned)
	}
	if failed > 0 {
		fmt.Printf("\033[31m✗ %d failed\033[0m", failed)
	}
	fmt.Println()

	if failed > 0 && !doctorFix {
		fmt.Println("\n\033[33mRun 'dev-cli doctor --fix' to attempt auto-fixes\033[0m")
		os.Exit(1)
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "ok":
		return "\033[32m✓\033[0m"
	case "warn":
		return "\033[33m⚠\033[0m"
	case "fail":
		return "\033[31m✗\033[0m"
	default:
		return "?"
	}
}

func attemptFix(result CheckResult) error {
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

func checkDocker() CheckResult {
	result := CheckResult{Name: "Docker"}

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

	// Get version
	versionCmd := exec.Command("docker", "--version")
	versionOutput, _ := versionCmd.Output()
	version := strings.TrimSpace(string(versionOutput))

	result.Status = "ok"
	result.Message = version
	return result
}

func checkDockerCompose() CheckResult {
	result := CheckResult{Name: "Docker Compose"}

	// Check docker compose (plugin)
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		cmd := exec.Command("docker", "compose", "version", "--short")
		output, _ := cmd.Output()
		result.Status = "ok"
		result.Message = "Plugin v" + strings.TrimSpace(string(output))
		return result
	}

	// Check docker-compose (standalone)
	if _, err := exec.LookPath("docker-compose"); err == nil {
		cmd := exec.Command("docker-compose", "--version")
		output, _ := cmd.Output()
		result.Status = "ok"
		result.Message = strings.TrimSpace(string(output))
		return result
	}

	// Neither available
	return CheckResult{
		Name:    "Docker Compose",
		Status:  "fail",
		Message: "Docker Compose not installed",
		FixCmd:  "sudo pacman -S docker-compose",
		FixFunc: func() error {
			// Try to install docker-compose via pacman (Arch)
			cmd := exec.Command("sudo", "pacman", "-S", "--noconfirm", "docker-compose")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
	}
}

func checkOllama() CheckResult {
	result := CheckResult{Name: "Ollama"}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")

	if err == nil && resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			result.Status = "ok"
			result.Message = "Running on localhost:11434"
			return result
		}
	}

	// Ollama not responding - check if Docker container exists
	dockerCheck := exec.Command("docker", "ps", "-a", "--filter", "name=ollama", "--format", "{{.Names}}")
	output, _ := dockerCheck.Output()
	containerExists := strings.TrimSpace(string(output)) != ""

	if containerExists {
		// Container exists but not responding - try starting it
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

	// Check if infra/ollama/docker-compose.yml exists
	projectRoot := getProjectRoot()
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

	// Fallback: check if native ollama is installed
	if _, err := exec.LookPath("ollama"); err == nil {
		return CheckResult{
			Name:    "Ollama",
			Status:  "fail",
			Message: "Ollama installed but not running",
			FixCmd:  "ollama serve &",
		}
	}

	// Nothing available - suggest Docker setup
	return CheckResult{
		Name:    "Ollama",
		Status:  "fail",
		Message: "Ollama not installed",
		FixCmd:  "cd infra/ollama && docker compose up -d",
		FixFunc: func() error {
			// Create infra/ollama if needed and start
			if projectRoot != "" {
				composeFile := filepath.Join(projectRoot, "infra", "ollama", "docker-compose.yml")
				if _, err := os.Stat(composeFile); err == nil {
					return runDockerCompose(composeFile, "up", "-d")
				}
			}
			return fmt.Errorf("docker-compose.yml not found - run from project root")
		},
	}
}

func checkOllamaModel() CheckResult {
	result := CheckResult{Name: "Ollama Model"}

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

	// Check if any model is available
	cmd := exec.Command("sh", "-c", "curl -s http://localhost:11434/api/tags | grep -o '\"name\":\"[^\"]*\"' | head -1")
	output, _ := cmd.Output()

	if len(output) == 0 || !strings.Contains(string(output), "name") {
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

	result.Status = "ok"
	result.Message = "Model(s) available"
	return result
}

func checkGPU() CheckResult {
	result := CheckResult{Name: "GPU (NVIDIA)"}

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
	result.Status = "ok"
	result.Message = gpuInfo
	return result
}

func checkDevlogsDir() CheckResult {
	result := CheckResult{Name: "Devlogs Directory"}

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

	result.Status = "ok"
	result.Message = devlogsDir
	return result
}

func checkNetwork() CheckResult {
	result := CheckResult{Name: "Network"}

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

	result.Status = "ok"
	result.Message = "External APIs reachable"
	return result
}

func getProjectRoot() string {
	// Try to find project root by looking for go.mod
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

// getDockerComposeCmd returns the correct docker compose command
// Returns ("docker-compose", []string{}) or ("docker", []string{"compose"})
func getDockerComposeCmd() (string, []string) {
	// Try docker compose (plugin) first
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		return "docker", []string{"compose"}
	}
	// Fall back to docker-compose (standalone)
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return "docker-compose", []string{}
	}
	// Default to docker compose
	return "docker", []string{"compose"}
}

// runDockerCompose runs docker compose with the given args
func runDockerCompose(composeFile string, args ...string) error {
	bin, prefix := getDockerComposeCmd()
	cmdArgs := append(prefix, "-f", composeFile)
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(bin, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
