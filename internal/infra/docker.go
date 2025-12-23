package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
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
	EnvVars   []string
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

type ImageInfo struct {
	ID      string
	Tags    []string
	Size    int64
	Created time.Time
}

type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	CreatedAt  time.Time
}

type ProcessInfo struct {
	PID     string
	User    string
	CPU     string
	Memory  string
	Command string
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

var (
	sharedDockerClient *DockerClient
	dockerClientOnce   sync.Once
	dockerClientErr    error
	dockerClientMu     sync.RWMutex
)

func GetSharedDockerClient() (*DockerClient, error) {
	dockerClientOnce.Do(func() {
		sharedDockerClient, dockerClientErr = NewDockerClient()
	})

	if dockerClientErr != nil {
		return nil, dockerClientErr
	}

	dockerClientMu.RLock()
	c := sharedDockerClient
	dockerClientMu.RUnlock()

	if c != nil && c.cli != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := c.cli.Ping(ctx); err == nil {
			return c, nil
		}

		dockerClientMu.Lock()
		if sharedDockerClient != nil {
			sharedDockerClient.Close()
		}
		sharedDockerClient, dockerClientErr = NewDockerClient()
		dockerClientMu.Unlock()
	}

	return sharedDockerClient, dockerClientErr
}

func ResetSharedDockerClient() {
	dockerClientMu.Lock()
	defer dockerClientMu.Unlock()
	if sharedDockerClient != nil {
		sharedDockerClient.Close()
		sharedDockerClient = nil
	}
	dockerClientOnce = sync.Once{}
	dockerClientErr = nil
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

		var ports []PortMapping
		for _, p := range c.Ports {
			ports = append(ports, PortMapping{
				Private:  p.PrivatePort,
				Public:   p.PublicPort,
				Protocol: p.Type,
				HostIP:   p.IP,
			})
		}

		health.Containers = append(health.Containers, ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Ports:   ports,
			Created: time.Unix(c.Created, 0),
		})
	}

	health.Available = true
	return health
}

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

	var lines []string
	buf := make([]byte, 8192)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data := buf[:n]
			for len(data) > 8 {
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

// Container Control Methods

func (d *DockerClient) StartContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (d *DockerClient) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return d.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (d *DockerClient) RestartContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return d.cli.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (d *DockerClient) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return d.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
}

func (d *DockerClient) KillContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerKill(ctx, containerID, "SIGKILL")
}

func (d *DockerClient) PauseContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerPause(ctx, containerID)
}

func (d *DockerClient) UnpauseContainer(ctx context.Context, containerID string) error {
	return d.cli.ContainerUnpause(ctx, containerID)
}

// Stats Streaming

func (d *DockerClient) GetContainerStats(ctx context.Context, containerID string) (*ContainerStatsSnapshot, error) {
	stats, err := d.cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("get stats failed: %w", err)
	}
	defer stats.Body.Close()

	// Define inline struct matching Docker stats JSON response
	var v struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  uint64 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64            `json:"usage"`
			Limit uint64            `json:"limit"`
			Stats map[string]uint64 `json:"stats"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
		BlkioStats struct {
			IoServiceBytesRecursive []struct {
				Op    string `json:"op"`
				Value uint64 `json:"value"`
			} `json:"io_service_bytes_recursive"`
		} `json:"blkio_stats"`
		PidsStats struct {
			Current uint64 `json:"current"`
		} `json:"pids_stats"`
	}

	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("decode stats failed: %w", err)
	}

	snapshot := &ContainerStatsSnapshot{
		Timestamp: time.Now(),
		PIDs:      v.PidsStats.Current,
	}

	// Calculate CPU percentage
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	if systemDelta > 0 && cpuDelta > 0 {
		snapshot.CPUPercent = (cpuDelta / systemDelta) * float64(v.CPUStats.OnlineCPUs) * 100.0
	}

	// Memory
	cacheVal := uint64(0)
	if v.MemoryStats.Stats != nil {
		cacheVal = v.MemoryStats.Stats["cache"]
	}
	snapshot.MemUsed = v.MemoryStats.Usage - cacheVal
	snapshot.MemLimit = v.MemoryStats.Limit
	if snapshot.MemLimit > 0 {
		snapshot.MemPercent = float64(snapshot.MemUsed) / float64(snapshot.MemLimit) * 100.0
	}

	// Network I/O
	for _, netStats := range v.Networks {
		snapshot.NetRx += netStats.RxBytes
		snapshot.NetTx += netStats.TxBytes
	}

	// Block I/O
	for _, bioEntry := range v.BlkioStats.IoServiceBytesRecursive {
		switch bioEntry.Op {
		case "Read", "read":
			snapshot.BlockRead += bioEntry.Value
		case "Write", "write":
			snapshot.BlockWrite += bioEntry.Value
		}
	}

	return snapshot, nil
}

// Container Inspection

func (d *DockerClient) InspectContainer(ctx context.Context, containerID string) (*ContainerDetail, error) {
	info, err := d.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect failed: %w", err)
	}

	name := info.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	createdTime, _ := time.Parse(time.RFC3339Nano, info.Created)

	detail := &ContainerDetail{
		ContainerInfo: ContainerInfo{
			ID:      info.ID[:12],
			Name:    name,
			Image:   info.Config.Image,
			Status:  info.State.Status,
			State:   info.State.Status,
			Created: createdTime,
		},
		EnvVars: info.Config.Env,
		Cmd:     info.Config.Cmd,
	}

	// Calculate uptime
	if info.State.Running {
		startTime, _ := time.Parse(time.RFC3339Nano, info.State.StartedAt)
		detail.Uptime = time.Since(startTime).Round(time.Second).String()
	}

	// Mounts
	for _, m := range info.Mounts {
		detail.Mounts = append(detail.Mounts, Mount{
			Source:      m.Source,
			Destination: m.Destination,
			Type:        string(m.Type),
			ReadOnly:    !m.RW,
		})
	}

	// Network
	for netName := range info.NetworkSettings.Networks {
		detail.NetworkID = netName
		break
	}

	// Ports
	for portProto, bindings := range info.NetworkSettings.Ports {
		for _, b := range bindings {
			var publicPort uint16
			if b.HostPort != "" {
				fmt.Sscanf(b.HostPort, "%d", &publicPort)
			}
			detail.Ports = append(detail.Ports, PortMapping{
				Private:  uint16(portProto.Int()),
				Public:   publicPort,
				Protocol: portProto.Proto(),
				HostIP:   b.HostIP,
			})
		}
	}

	return detail, nil
}

// Images

func (d *DockerClient) ListImages(ctx context.Context) ([]ImageInfo, error) {
	images, err := d.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list images failed: %w", err)
	}

	var result []ImageInfo
	for _, img := range images {
		id := img.ID
		if len(id) > 19 {
			id = id[7:19] // Remove "sha256:" prefix and truncate
		}
		result = append(result, ImageInfo{
			ID:      id,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0),
		})
	}
	return result, nil
}

func (d *DockerClient) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := d.cli.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	return err
}

// Volumes

func (d *DockerClient) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	volumes, err := d.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list volumes failed: %w", err)
	}

	var result []VolumeInfo
	for _, vol := range volumes.Volumes {
		createdAt, _ := time.Parse(time.RFC3339, vol.CreatedAt)
		result = append(result, VolumeInfo{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			CreatedAt:  createdAt,
		})
	}
	return result, nil
}

func (d *DockerClient) RemoveVolume(ctx context.Context, volumeName string, force bool) error {
	return d.cli.VolumeRemove(ctx, volumeName, force)
}

// Container Processes (docker top)

func (d *DockerClient) TopContainer(ctx context.Context, containerID string) ([]ProcessInfo, error) {
	top, err := d.cli.ContainerTop(ctx, containerID, []string{})
	if err != nil {
		return nil, fmt.Errorf("top failed: %w", err)
	}

	// Find column indices
	pidIdx, userIdx, cmdIdx := -1, -1, -1
	for i, title := range top.Titles {
		switch title {
		case "PID":
			pidIdx = i
		case "USER":
			userIdx = i
		case "CMD", "COMMAND":
			cmdIdx = i
		}
	}

	var result []ProcessInfo
	for _, proc := range top.Processes {
		info := ProcessInfo{}
		if pidIdx >= 0 && pidIdx < len(proc) {
			info.PID = proc[pidIdx]
		}
		if userIdx >= 0 && userIdx < len(proc) {
			info.User = proc[userIdx]
		}
		if cmdIdx >= 0 && cmdIdx < len(proc) {
			info.Command = proc[cmdIdx]
		}
		result = append(result, info)
	}
	return result, nil
}

// Bulk operations

func (d *DockerClient) PruneContainers(ctx context.Context) (uint64, error) {
	report, err := d.cli.ContainersPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

func (d *DockerClient) PruneImages(ctx context.Context) (uint64, error) {
	report, err := d.cli.ImagesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

func (d *DockerClient) PruneVolumes(ctx context.Context) (uint64, error) {
	report, err := d.cli.VolumesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}
