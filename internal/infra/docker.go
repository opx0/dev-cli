package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"dev-cli/internal/executor"
)

type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	Ports   []PortMapping
	Created time.Time
}

type PortMapping struct {
	Private  uint16
	Public   uint16
	Protocol string
	HostIP   string
}

type ContainerDetail struct {
	ContainerInfo
	Mounts    []Mount
	NetworkID string
	Cmd       []string
	Uptime    string
}

type Mount struct {
	Source      string
	Destination string
	Type        string
	ReadOnly    bool
}

type ContainerStatsSnapshot struct {
	CPUPercent float64
	MemUsed    uint64
	MemLimit   uint64
	MemPercent float64
	NetRx      uint64
	NetTx      uint64
	BlockRead  uint64
	BlockWrite uint64
	PIDs       uint64
	Timestamp  time.Time
}

type DockerHealth struct {
	Available  bool
	Version    string
	Containers []ContainerInfo
	Error      error
}

type DockerClient struct{ binary string }

func NewDockerClient() (*DockerClient, error) {
	binary, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker executable not found: %w", err)
	}
	return &DockerClient{binary: binary}, nil
}

func (d *DockerClient) CheckHealth(ctx context.Context) DockerHealth {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	version := executor.ExecuteProgram(checkCtx, "", d.binary, "version", "--format", "{{.Server.Version}}")
	if version.ExitCode != 0 {
		return DockerHealth{Error: fmt.Errorf("daemon unavailable: %s", version.Output)}
	}

	list := executor.ExecuteProgram(checkCtx, "", d.binary, "ps", "-a", "--format", "{{json .}}")
	if list.ExitCode != 0 {
		return DockerHealth{Error: fmt.Errorf("container list failed: %s", list.Output)}
	}

	health := DockerHealth{Available: true, Version: strings.TrimSpace(version.Output)}
	for _, line := range strings.Split(list.Output, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var item struct {
			ID        string `json:"ID"`
			Names     string `json:"Names"`
			Image     string `json:"Image"`
			Status    string `json:"Status"`
			State     string `json:"State"`
			CreatedAt string `json:"CreatedAt"`
		}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return DockerHealth{Error: fmt.Errorf("decode container list: %w", err)}
		}
		created, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", item.CreatedAt)
		health.Containers = append(health.Containers, ContainerInfo{
			ID: item.ID, Name: item.Names, Image: item.Image, Status: item.Status,
			State: item.State, Created: created,
		})
	}
	return health
}

func (d *DockerClient) GetContainerLogs(ctx context.Context, containerID string, tail int) ([]string, error) {
	result := executor.ExecuteProgram(ctx, "", d.binary, "logs", "--timestamps", "--tail", strconv.Itoa(tail), containerID)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("get logs failed: %s", result.Output)
	}
	text := strings.TrimSpace(result.Output)
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

func (d *DockerClient) GetContainerStats(ctx context.Context, containerID string) (*ContainerStatsSnapshot, error) {
	format := "{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}"
	result := executor.ExecuteProgram(ctx, "", d.binary, "stats", "--no-stream", "--format", format, containerID)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("get stats failed: %s", result.Output)
	}
	fields := strings.Split(strings.TrimSpace(result.Output), "\t")
	if len(fields) != 6 {
		return nil, fmt.Errorf("unexpected docker stats output")
	}

	snapshot := &ContainerStatsSnapshot{Timestamp: time.Now()}
	snapshot.CPUPercent, _ = strconv.ParseFloat(strings.TrimSuffix(fields[0], "%"), 64)
	snapshot.MemUsed, snapshot.MemLimit = parseBytePair(fields[1])
	snapshot.MemPercent, _ = strconv.ParseFloat(strings.TrimSuffix(fields[2], "%"), 64)
	snapshot.NetRx, snapshot.NetTx = parseBytePair(fields[3])
	snapshot.BlockRead, snapshot.BlockWrite = parseBytePair(fields[4])
	snapshot.PIDs, _ = strconv.ParseUint(strings.TrimSpace(fields[5]), 10, 64)
	return snapshot, nil
}

func (d *DockerClient) InspectContainer(ctx context.Context, containerID string) (*ContainerDetail, error) {
	result := executor.ExecuteProgram(ctx, "", d.binary, "inspect", containerID)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("inspect failed: %s", result.Output)
	}

	var response []struct {
		ID      string `json:"Id"`
		Name    string `json:"Name"`
		Created string `json:"Created"`
		Config  struct {
			Image string   `json:"Image"`
			Cmd   []string `json:"Cmd"`
		} `json:"Config"`
		State struct {
			Status    string `json:"Status"`
			Running   bool   `json:"Running"`
			StartedAt string `json:"StartedAt"`
		} `json:"State"`
		Mounts []struct {
			Source      string `json:"Source"`
			Destination string `json:"Destination"`
			Type        string `json:"Type"`
			RW          bool   `json:"RW"`
		} `json:"Mounts"`
		NetworkSettings struct {
			Networks map[string]json.RawMessage `json:"Networks"`
			Ports    map[string][]struct {
				HostIP   string `json:"HostIp"`
				HostPort string `json:"HostPort"`
			} `json:"Ports"`
		} `json:"NetworkSettings"`
	}
	if err := json.Unmarshal([]byte(result.Output), &response); err != nil || len(response) != 1 {
		if err == nil {
			err = fmt.Errorf("expected one container")
		}
		return nil, fmt.Errorf("decode inspect output: %w", err)
	}

	item := response[0]
	created, _ := time.Parse(time.RFC3339Nano, item.Created)
	detail := &ContainerDetail{ContainerInfo: ContainerInfo{
		ID: item.ID, Name: strings.TrimPrefix(item.Name, "/"), Image: item.Config.Image,
		Status: item.State.Status, State: item.State.Status, Created: created,
	}, Cmd: item.Config.Cmd}
	if item.State.Running {
		if started, err := time.Parse(time.RFC3339Nano, item.State.StartedAt); err == nil {
			detail.Uptime = time.Since(started).Round(time.Second).String()
		}
	}
	for _, mount := range item.Mounts {
		detail.Mounts = append(detail.Mounts, Mount{
			Source: mount.Source, Destination: mount.Destination, Type: mount.Type, ReadOnly: !mount.RW,
		})
	}
	for name := range item.NetworkSettings.Networks {
		detail.NetworkID = name
		break
	}
	for privatePort, bindings := range item.NetworkSettings.Ports {
		parts := strings.SplitN(privatePort, "/", 2)
		private, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			continue
		}
		protocol := "tcp"
		if len(parts) == 2 {
			protocol = parts[1]
		}
		for _, binding := range bindings {
			public, err := strconv.ParseUint(binding.HostPort, 10, 16)
			if err != nil {
				continue
			}
			detail.Ports = append(detail.Ports, PortMapping{
				Private: uint16(private), Public: uint16(public), Protocol: protocol, HostIP: binding.HostIP,
			})
		}
	}
	return detail, nil
}

func parseBytePair(value string) (uint64, uint64) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0
	}
	return parseBytes(parts[0]), parseBytes(parts[1])
}

func parseBytes(value string) uint64 {
	value = strings.TrimSpace(value)
	index := 0
	for index < len(value) && ((value[index] >= '0' && value[index] <= '9') || value[index] == '.') {
		index++
	}
	if index == 0 {
		return 0
	}
	number, err := strconv.ParseFloat(value[:index], 64)
	if err != nil || number < 0 {
		return 0
	}
	multipliers := map[string]float64{
		"b": 1, "kb": 1e3, "kib": 1 << 10, "mb": 1e6, "mib": 1 << 20,
		"gb": 1e9, "gib": 1 << 30, "tb": 1e12, "tib": 1 << 40,
	}
	return uint64(number * multipliers[strings.ToLower(strings.TrimSpace(value[index:]))])
}
