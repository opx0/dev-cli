package infra

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestIntegration_DockerClient_CheckHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "alpine:latest",
		Cmd:          []string{"sleep", "30"},
		WaitingFor:   wait.ForLog("").WithStartupTimeout(10 * time.Second),
		ExposedPorts: []string{},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer container.Terminate(ctx)

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}
	defer client.Close()

	health := client.CheckHealth(ctx)
	if !health.Available {
		t.Fatalf("docker should be available: %v", health.Error)
	}

	if len(health.Containers) == 0 {
		t.Error("expected at least one container to be visible")
	}

	containerID := container.GetContainerID()

	found := false
	for _, c := range health.Containers {
		if c.ID == containerID || c.ID[:12] == containerID[:12] {
			found = true
			if c.State != "running" {
				t.Errorf("expected container state 'running', got '%s'", c.State)
			}
			break
		}
	}

	if !found {
		t.Error("test container not found in health check")
	}
}

func TestIntegration_DockerClient_StartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:      "alpine:latest",
		Cmd:        []string{"sleep", "60"},
		WaitingFor: wait.ForLog("").WithStartupTimeout(10 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer container.Terminate(ctx)

	containerID := container.GetContainerID()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}
	defer client.Close()

	if err := client.StopContainer(ctx, containerID); err != nil {
		t.Fatalf("failed to stop container: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	health := client.CheckHealth(ctx)
	for _, c := range health.Containers {
		if c.ID == containerID || c.ID[:12] == containerID[:12] {
			if c.State == "running" {
				t.Error("container should not be running after stop")
			}
			break
		}
	}

	if err := client.StartContainer(ctx, containerID); err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	health = client.CheckHealth(ctx)
	for _, c := range health.Containers {
		if c.ID == containerID || c.ID[:12] == containerID[:12] {
			if c.State != "running" {
				t.Errorf("container should be running after start, got '%s'", c.State)
			}
			break
		}
	}
}

func TestIntegration_DockerClient_ContainerLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:      "alpine:latest",
		Cmd:        []string{"sh", "-c", "echo 'test-log-output' && sleep 5"},
		WaitingFor: wait.ForLog("test-log-output").WithStartupTimeout(10 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	defer container.Terminate(ctx)

	containerID := container.GetContainerID()

	client, err := NewDockerClient()
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}
	defer client.Close()

	time.Sleep(1 * time.Second)

	logs, err := client.GetContainerLogs(ctx, containerID, 10)
	if err != nil {
		t.Fatalf("failed to get container logs: %v", err)
	}

	if len(logs) == 0 {
		t.Error("expected at least one log line")
	}

	found := false
	for _, line := range logs {
		if line == "test-log-output" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expected log containing 'test-log-output', got: %v", logs)
	}
}

func TestIntegration_MockOllamaAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	nginxConf := `
events {}
http {
    server {
        listen 11434;
        location /api/tags {
            default_type application/json;
            return 200 '{"models":[{"name":"qwen2.5-coder:3b-instruct","modified_at":"2024-01-01T00:00:00Z","size":1234567890,"digest":"abc123"}]}';
        }
    }
}
`

	req := testcontainers.ContainerRequest{
		Image:        "nginx:alpine",
		ExposedPorts: []string{"11434/tcp"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      "",
				Reader:            nil,
				ContainerFilePath: "/etc/nginx/nginx.conf",
			},
		},
		Cmd:        []string{"sh", "-c", fmt.Sprintf("echo '%s' > /etc/nginx/nginx.conf && nginx -g 'daemon off;'", nginxConf)},
		WaitingFor: wait.ForHTTP("/api/tags").WithPort("11434/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start mock ollama container: %v", err)
	}
	defer container.Terminate(ctx)

	mappedPort, err := container.MappedPort(ctx, "11434")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get host: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, mappedPort.Port())

	ollamaClient := NewOllamaClient(nil, baseURL)

	if err := ollamaClient.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}

	models, err := ollamaClient.ListModels(ctx)
	if err != nil {
		t.Fatalf("list models failed: %v", err)
	}

	if len(models) != 1 {
		t.Errorf("expected 1 model, got %d", len(models))
	}

	if len(models) > 0 && models[0].Name != "qwen2.5-coder:3b-instruct" {
		t.Errorf("expected model name 'qwen2.5-coder:3b-instruct', got '%s'", models[0].Name)
	}

	hasModel, err := ollamaClient.HasModel(ctx, "qwen2.5-coder")
	if err != nil {
		t.Fatalf("has model failed: %v", err)
	}

	if !hasModel {
		t.Error("expected HasModel to return true for 'qwen2.5-coder'")
	}

	hasModel, err = ollamaClient.HasModel(ctx, "nonexistent-model")
	if err != nil {
		t.Fatalf("has model failed: %v", err)
	}

	if hasModel {
		t.Error("expected HasModel to return false for 'nonexistent-model'")
	}
}
