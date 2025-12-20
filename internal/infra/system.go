package infra

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type GPUStats struct {
	Available      bool
	UsedMemoryMB   int
	TotalMemoryMB  int
	UtilizationPct int
	Error          error
}

func GetGPUStats() GPUStats {
	stats := GPUStats{}

	cmd := exec.Command("nvidia-smi", "--query-gpu=memory.used,memory.total", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		stats.Available = false
		stats.Error = err
		return stats
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) != 2 {
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

	return stats
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

type StarshipPrompt struct {
	Available bool
	Raw       string   // Raw prompt output
	Clean     string   // ANSI-stripped version
	Segments  []string // Parsed segments
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
