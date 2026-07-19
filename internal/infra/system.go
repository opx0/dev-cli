package infra

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type PortConflict struct {
	Port      int
	Process   string
	PID       int
	Suggested int
}

func CheckPortAvailable(ctx context.Context, port int) *PortConflict {
	dialer := net.Dialer{Timeout: 200 * time.Millisecond}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return nil
	}
	_ = conn.Close()
	pid, process, _ := GetProcessOnPort(ctx, port)
	return &PortConflict{Port: port, Process: process, PID: pid, Suggested: FindAvailablePort(ctx, port+1)}
}

func FindAvailablePort(ctx context.Context, basePort int) int {
	listenConfig := net.ListenConfig{}
	for port := basePort; port < basePort+100; port++ {
		listener, err := listenConfig.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			_ = listener.Close()
			return port
		}
	}
	return 0
}

func GetProcessOnPort(ctx context.Context, port int) (pid int, process string, err error) {
	// #nosec G204 -- port is validated as 1..65535 by the caller and is one argument.
	output, err := exec.CommandContext(ctx, "lsof", "-i", fmt.Sprintf(":%d", port), "-t").Output()
	if err != nil {
		return 0, "", err
	}
	line := strings.Split(strings.TrimSpace(string(output)), "\n")[0]
	pid, err = strconv.Atoi(line)
	if err != nil {
		return 0, "", err
	}
	if runtime.GOOS == "linux" {
		if command, readErr := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); readErr == nil {
			process = strings.TrimSpace(string(command))
		}
	} else {
		// #nosec G204 -- pid was parsed as an integer from lsof output.
		command, commandErr := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
		if commandErr == nil {
			process = strings.TrimSpace(string(command))
		}
	}
	return pid, process, nil
}
