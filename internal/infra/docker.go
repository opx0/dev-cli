package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	Status string
	State  string
}

type DockerHealth struct {
	Available  bool
	Version    string
	Containers []ContainerInfo
	Error      error
}

type DockerClient struct {
	cli *client.Client
}

func NewDockerClient() (*DockerClient, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client failed: %w", err)
	}
	return &DockerClient{cli: cli}, nil
}

func (d *DockerClient) CheckHealth(ctx context.Context) DockerHealth {
	health := DockerHealth{}

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := d.cli.Ping(checkCtx)
	if err != nil {
		health.Error = fmt.Errorf("daemon unavailable: %w", err)
		return health
	}

	version, err := d.cli.ServerVersion(checkCtx)
	if err != nil {
		health.Error = fmt.Errorf("version check failed: %w", err)
		return health
	}
	health.Version = version.Version

	containers, err := d.cli.ContainerList(checkCtx, container.ListOptions{All: true})
	if err != nil {
		health.Error = fmt.Errorf("container list failed: %w", err)
		return health
	}

	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		health.Containers = append(health.Containers, ContainerInfo{
			ID:     c.ID[:12],
			Name:   name,
			Image:  c.Image,
			Status: c.Status,
			State:  c.State,
		})
	}

	health.Available = true
	return health
}

// GetContainerLogs fetches logs from a container
func (d *DockerClient) GetContainerLogs(ctx context.Context, containerID string, tail int) ([]string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
		Timestamps: true,
	}

	reader, err := d.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("get logs failed: %w", err)
	}
	defer reader.Close()

	// Read logs
	var lines []string
	buf := make([]byte, 8192)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			// Docker logs have an 8-byte header per line
			data := buf[:n]
			for len(data) > 8 {
				// Skip header (8 bytes)
				lineEnd := 8
				for lineEnd < len(data) && data[lineEnd] != '\n' {
					lineEnd++
				}
				if lineEnd > 8 {
					line := string(data[8:lineEnd])
					lines = append(lines, line)
				}
				if lineEnd >= len(data) {
					break
				}
				data = data[lineEnd+1:]
			}
		}
		if err != nil {
			break
		}
	}

	return lines, nil
}

func (d *DockerClient) Close() error {
	if d.cli != nil {
		return d.cli.Close()
	}
	return nil
}
