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

// --- Service checks ---

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

// --- Port utilities ---

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

// --- Starship prompt integration ---

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
