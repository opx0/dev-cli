package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dev-cli/internal/infra"
)

// QueryDockerTool queries Docker containers for logs, stats, and inspection.
type QueryDockerTool struct{}

func (t *QueryDockerTool) Name() string { return "query_docker" }
func (t *QueryDockerTool) Description() string {
	return "Query Docker containers for logs, stats, and info"
}

func (t *QueryDockerTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "action", Type: "string", Description: "Action: logs, stats, inspect, list", Required: true},
		{Name: "container", Type: "string", Description: "Container ID or name", Required: false},
		{Name: "tail", Type: "int", Description: "Number of log lines (for logs action)", Required: false, Default: 100},
	}
}

// DockerLogsResult contains container logs output.
type DockerLogsResult struct {
	Container string   `json:"container"`
	Lines     []string `json:"lines"`
	Count     int      `json:"count"`
}

// DockerStatsResult contains container stats.
type DockerStatsResult struct {
	Container  string  `json:"container"`
	CPUPercent float64 `json:"cpu_percent"`
	MemUsedMB  uint64  `json:"mem_used_mb"`
	MemLimitMB uint64  `json:"mem_limit_mb"`
	MemPercent float64 `json:"mem_percent"`
	NetRxMB    float64 `json:"net_rx_mb"`
	NetTxMB    float64 `json:"net_tx_mb"`
	PIDs       uint64  `json:"pids"`
}

// DockerInspectResult contains container inspection details.
type DockerInspectResult struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	State     string   `json:"state"`
	Status    string   `json:"status"`
	Ports     []string `json:"ports"`
	Mounts    []string `json:"mounts"`
	EnvVars   []string `json:"env_vars"`
	Cmd       []string `json:"cmd"`
	NetworkID string   `json:"network_id"`
	Uptime    string   `json:"uptime"`
}

// DockerListResult contains list of containers.
type DockerListResult struct {
	Containers []DockerContainerInfo `json:"containers"`
	Count      int                   `json:"count"`
}

// DockerContainerInfo contains basic container info.
type DockerContainerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	State  string `json:"state"`
	Status string `json:"status"`
}

func (t *QueryDockerTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	action := GetString(params, "action", "")
	if action == "" {
		return NewErrorResult("action is required (logs, stats, inspect, list)", time.Since(start))
	}

	docker, err := infra.GetRegistry().Docker()
	if err != nil {
		return NewErrorResult(fmt.Sprintf("Docker not available: %v", err), time.Since(start))
	}

	switch action {
	case "logs":
		return t.getLogs(ctx, docker, params, start)
	case "stats":
		return t.getStats(ctx, docker, params, start)
	case "inspect":
		return t.inspect(ctx, docker, params, start)
	case "list":
		return t.list(ctx, docker, start)
	default:
		return NewErrorResult(fmt.Sprintf("unknown action: %s", action), time.Since(start))
	}
}

func (t *QueryDockerTool) getLogs(ctx context.Context, docker *infra.DockerClient, params map[string]any, start time.Time) ToolResult {
	container := GetString(params, "container", "")
	if container == "" {
		return NewErrorResult("container is required for logs action", time.Since(start))
	}

	tail := GetInt(params, "tail", 100)

	lines, err := docker.GetContainerLogs(ctx, container, tail)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to get logs: %v", err), time.Since(start))
	}

	return NewResult(DockerLogsResult{
		Container: container,
		Lines:     lines,
		Count:     len(lines),
	}, time.Since(start))
}

func (t *QueryDockerTool) getStats(ctx context.Context, docker *infra.DockerClient, params map[string]any, start time.Time) ToolResult {
	container := GetString(params, "container", "")
	if container == "" {
		return NewErrorResult("container is required for stats action", time.Since(start))
	}

	stats, err := docker.GetContainerStats(ctx, container)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to get stats: %v", err), time.Since(start))
	}

	return NewResult(DockerStatsResult{
		Container:  container,
		CPUPercent: stats.CPUPercent,
		MemUsedMB:  stats.MemUsed / (1024 * 1024),
		MemLimitMB: stats.MemLimit / (1024 * 1024),
		MemPercent: stats.MemPercent,
		NetRxMB:    float64(stats.NetRx) / (1024 * 1024),
		NetTxMB:    float64(stats.NetTx) / (1024 * 1024),
		PIDs:       stats.PIDs,
	}, time.Since(start))
}

func (t *QueryDockerTool) inspect(ctx context.Context, docker *infra.DockerClient, params map[string]any, start time.Time) ToolResult {
	container := GetString(params, "container", "")
	if container == "" {
		return NewErrorResult("container is required for inspect action", time.Since(start))
	}

	detail, err := docker.InspectContainer(ctx, container)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to inspect: %v", err), time.Since(start))
	}

	ports := make([]string, 0, len(detail.Ports))
	for _, p := range detail.Ports {
		ports = append(ports, fmt.Sprintf("%d:%d/%s", p.Public, p.Private, p.Protocol))
	}

	mounts := make([]string, 0, len(detail.Mounts))
	for _, m := range detail.Mounts {
		mounts = append(mounts, fmt.Sprintf("%s:%s", m.Source, m.Destination))
	}

	return NewResult(DockerInspectResult{
		ID:        detail.ID,
		Name:      detail.Name,
		Image:     detail.Image,
		State:     detail.State,
		Status:    detail.Status,
		Ports:     ports,
		Mounts:    mounts,
		EnvVars:   detail.EnvVars,
		Cmd:       detail.Cmd,
		NetworkID: detail.NetworkID,
		Uptime:    detail.Uptime,
	}, time.Since(start))
}

func (t *QueryDockerTool) list(ctx context.Context, docker *infra.DockerClient, start time.Time) ToolResult {
	health := docker.CheckHealth(ctx)
	if !health.Available {
		return NewErrorResult("Docker not available", time.Since(start))
	}

	containers := make([]DockerContainerInfo, 0, len(health.Containers))
	for _, c := range health.Containers {
		containers = append(containers, DockerContainerInfo{
			ID:     c.ID[:12],
			Name:   strings.TrimPrefix(c.Name, "/"),
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
		})
	}

	return NewResult(DockerListResult{
		Containers: containers,
		Count:      len(containers),
	}, time.Since(start))
}
