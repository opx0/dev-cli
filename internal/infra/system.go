package infra

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type GPUProvider interface {
	GetStats() GPUStats
	Vendor() string
}

type GPUStats struct {
	Available      bool
	Vendor         string
	UsedMemoryMB   int
	TotalMemoryMB  int
	UtilizationPct int
	Temperature    int
	Error          error
}

func DetectGPU() GPUProvider {
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		return &NvidiaGPUProvider{}
	}

	if _, err := exec.LookPath("rocm-smi"); err == nil {
		return &AMDGPUProvider{}
	}

	if runtime.GOOS == "darwin" {
		return &AppleGPUProvider{}
	}

	return &NoGPUProvider{}
}

type NvidiaGPUProvider struct{}

func (p *NvidiaGPUProvider) Vendor() string {
	return "nvidia"
}

func (p *NvidiaGPUProvider) GetStats() GPUStats {
	stats := GPUStats{Vendor: "nvidia"}

	cmd := exec.Command("nvidia-smi",
		"--query-gpu=memory.used,memory.total,utilization.gpu,temperature.gpu",
		"--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 2 {
		stats.Available = false
		stats.Error = fmt.Errorf("unexpected output format")
		return stats
	}

	used, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	total, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))

	if err1 != nil || err2 != nil {
		stats.Available = false
		stats.Error = fmt.Errorf("failed to parse memory values")
		return stats
	}

	stats.Available = true
	stats.UsedMemoryMB = used
	stats.TotalMemoryMB = total
	if total > 0 {
		stats.UtilizationPct = (used * 100) / total
	}

	if len(parts) >= 3 {
		if util, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
			stats.UtilizationPct = util
		}
	}

	if len(parts) >= 4 {
		if temp, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil {
			stats.Temperature = temp
		}
	}

	return stats
}

type AMDGPUProvider struct{}

func (p *AMDGPUProvider) Vendor() string {
	return "amd"
}

func (p *AMDGPUProvider) GetStats() GPUStats {
	stats := GPUStats{Vendor: "amd"}

	cmd := exec.Command("rocm-smi", "--showmeminfo", "vram", "--csv")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "GPU") && strings.Contains(line, ",") {
			parts := strings.Split(line, ",")
			if len(parts) >= 3 {
				if used, err := parseMemoryMB(strings.TrimSpace(parts[1])); err == nil {
					stats.UsedMemoryMB = used
				}
				if total, err := parseMemoryMB(strings.TrimSpace(parts[2])); err == nil {
					stats.TotalMemoryMB = total
				}
			}
		}
	}

	tempCmd := exec.Command("rocm-smi", "-t", "--csv")
	tempOutput, err := tempCmd.Output()
	if err == nil {
		lines := strings.Split(string(tempOutput), "\n")
		for _, line := range lines {
			if strings.Contains(line, "GPU") {
				parts := strings.Split(line, ",")
				if len(parts) >= 2 {
					if temp, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
						stats.Temperature = int(temp)
					}
				}
			}
		}
	}

	stats.Available = stats.TotalMemoryMB > 0
	if stats.TotalMemoryMB > 0 {
		stats.UtilizationPct = (stats.UsedMemoryMB * 100) / stats.TotalMemoryMB
	}

	return stats
}

func parseMemoryMB(s string) (int, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")

	if strings.HasSuffix(s, "GB") || strings.HasSuffix(s, "G") {
		s = strings.TrimSuffix(strings.TrimSuffix(s, "GB"), "G")
		if val, err := strconv.ParseFloat(s, 64); err == nil {
			return int(val * 1024), nil
		}
	}
	if strings.HasSuffix(s, "MB") || strings.HasSuffix(s, "M") {
		s = strings.TrimSuffix(strings.TrimSuffix(s, "MB"), "M")
		if val, err := strconv.ParseFloat(s, 64); err == nil {
			return int(val), nil
		}
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val, nil
	}
	return 0, fmt.Errorf("cannot parse: %s", s)
}

type AppleGPUProvider struct{}

func (p *AppleGPUProvider) Vendor() string {
	return "apple"
}

func (p *AppleGPUProvider) GetStats() GPUStats {
	stats := GPUStats{Vendor: "apple"}

	cmd := exec.Command("system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	if strings.Contains(string(output), "Apple") || strings.Contains(string(output), "M1") ||
		strings.Contains(string(output), "M2") || strings.Contains(string(output), "M3") {
		stats.Available = true
	}

	return stats
}

type NoGPUProvider struct{}

func (p *NoGPUProvider) Vendor() string {
	return "none"
}

func (p *NoGPUProvider) GetStats() GPUStats {
	return GPUStats{
		Available: false,
		Vendor:    "none",
		Error:     fmt.Errorf("no supported GPU detected"),
	}
}

func GetGPUStats() GPUStats {
	return DetectGPU().GetStats()
}

type ServiceStatus struct {
	Name      string
	Port      int
	Available bool
	Error     error
}

func CheckServices() []ServiceStatus {
	services := []struct {
		name string
		port int
	}{
		{"Postgres", 5432},
		{"Redis", 6379},
		{"Ollama", 11434},
	}

	var results []ServiceStatus

	for _, s := range services {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", s.port), 500*time.Millisecond)
		status := ServiceStatus{
			Name: s.name,
			Port: s.port,
		}

		if err != nil {
			status.Available = false
			status.Error = err
		} else {
			status.Available = true
			conn.Close()
		}
		results = append(results, status)
	}

	return results
}

type PortConflict struct {
	Port      int
	Process   string
	PID       int
	Suggested int
}

func CheckPortAvailable(port int) *PortConflict {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		return nil
	}
	conn.Close()

	pid, process, _ := GetProcessOnPort(port)

	return &PortConflict{
		Port:      port,
		Process:   process,
		PID:       pid,
		Suggested: FindAvailablePort(port + 1),
	}
}

func FindAvailablePort(basePort int) int {
	for port := basePort; port < basePort+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return 0
}

func GetProcessOnPort(port int) (pid int, process string, err error) {
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-t")
	output, err := cmd.Output()
	if err != nil {
		return 0, "", err
	}

	pidStr := strings.TrimSpace(string(output))
	lines := strings.Split(pidStr, "\n")
	if len(lines) == 0 || lines[0] == "" {
		return 0, "", fmt.Errorf("no process found")
	}

	pid, err = strconv.Atoi(lines[0])
	if err != nil {
		return 0, "", err
	}

	if runtime.GOOS == "linux" {
		cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
		if err == nil {
			process = strings.TrimSpace(string(cmdline))
		}
	} else {
		psCmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=")
		psOutput, err := psCmd.Output()
		if err == nil {
			process = strings.TrimSpace(string(psOutput))
		}
	}

	return pid, process, nil
}

type StarshipPrompt struct {
	Available bool
	Raw       string
	Clean     string
	Segments  []string
}

func GetStarshipPrompt() StarshipPrompt {
	prompt := StarshipPrompt{}

	path, err := exec.LookPath("starship")
	if err != nil || path == "" {
		return prompt
	}
	prompt.Available = true

	cmd := exec.Command("starship", "prompt")
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return prompt
	}

	prompt.Raw = string(out)
	prompt.Clean = stripANSI(prompt.Raw)
	prompt.Segments = parseStarshipSegments(prompt.Clean)

	return prompt
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m|%\{[^}]*\}`)
	clean := re.ReplaceAllString(s, "")
	clean = strings.Join(strings.Fields(clean), " ")
	return strings.TrimSpace(clean)
}

func parseStarshipSegments(clean string) []string {
	separators := []string{" on ", " in ", " via ", " is ", " took "}

	parts := []string{clean}
	for _, sep := range separators {
		var newParts []string
		for _, part := range parts {
			split := strings.Split(part, sep)
			for i, s := range split {
				s = strings.TrimSpace(s)
				if s != "" {
					if i > 0 {
						newParts = append(newParts, sep[1:len(sep)-1]+" "+s)
					} else {
						newParts = append(newParts, s)
					}
				}
			}
		}
		parts = newParts
	}

	var filtered []string
	for _, p := range parts {
		p = strings.TrimLeft(p, "❯> ")
		p = strings.TrimSpace(p)
		if p != "" && p != "❯" && p != ">" {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

func GetStarshipStatusLine() string {
	prompt := GetStarshipPrompt()
	if !prompt.Available {
		return ""
	}

	line := prompt.Clean
	line = strings.TrimRight(line, "❯> \n\r")
	line = strings.TrimSpace(line)

	return line
}
