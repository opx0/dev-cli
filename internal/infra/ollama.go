package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
	docker     *DockerClient
}

type OllamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
	Digest     string    `json:"digest"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

type OllamaPullProgress struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

func NewOllamaClient(docker *DockerClient, baseURL string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		docker: docker,
	}
}

func (o *OllamaClient) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama not responding: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

func (o *OllamaClient) ListModels(ctx context.Context) ([]OllamaModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list models failed with status %d", resp.StatusCode)
	}

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return tagsResp.Models, nil
}

func (o *OllamaClient) HasModel(ctx context.Context, name string) (bool, error) {
	models, err := o.ListModels(ctx)
	if err != nil {
		return false, err
	}

	searchName := strings.ToLower(name)

	for _, model := range models {
		modelName := strings.ToLower(model.Name)
		if modelName == searchName || strings.HasPrefix(modelName, searchName+":") {
			return true, nil
		}
	}

	return false, nil
}

func (o *OllamaClient) PullModel(ctx context.Context, name string) (io.ReadCloser, error) {
	payload := fmt.Sprintf(`{"name": "%s", "stream": true}`, name)
	req, err := http.NewRequestWithContext(ctx, "POST", o.baseURL+"/api/pull", strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 0,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pull model: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("pull failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (o *OllamaClient) PullModelSync(ctx context.Context, name string, progressFn func(OllamaPullProgress)) error {
	reader, err := o.PullModel(ctx, name)
	if err != nil {
		return err
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	for {
		var progress OllamaPullProgress
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode progress: %w", err)
		}

		if progressFn != nil {
			progressFn(progress)
		}

		if progress.Status == "success" {
			break
		}
	}

	return nil
}

func (o *OllamaClient) EnsureModel(ctx context.Context, name string, progressFn func(OllamaPullProgress)) error {
	if err := o.EnsureContainer(ctx); err != nil {
		return fmt.Errorf("ensure container: %w", err)
	}

	if err := o.waitForAPI(ctx); err != nil {
		return fmt.Errorf("wait for API: %w", err)
	}

	hasModel, err := o.HasModel(ctx, name)
	if err != nil {
		return fmt.Errorf("check model: %w", err)
	}

	if hasModel {
		return nil
	}

	return o.PullModelSync(ctx, name, progressFn)
}

func (o *OllamaClient) EnsureContainer(ctx context.Context) error {
	if o.docker == nil {
		return fmt.Errorf("docker client not available")
	}

	health := o.docker.CheckHealth(ctx)
	if !health.Available {
		return fmt.Errorf("docker not available: %v", health.Error)
	}

	var ollamaContainer *ContainerInfo
	for i := range health.Containers {
		c := &health.Containers[i]
		if strings.Contains(strings.ToLower(c.Name), "ollama") ||
			strings.Contains(strings.ToLower(c.Image), "ollama") {
			ollamaContainer = c
			break
		}
	}

	if ollamaContainer == nil {
		return fmt.Errorf("ollama container not found - run 'docker compose up -d' in infra/ollama")
	}

	if ollamaContainer.State != "running" {
		if err := o.docker.StartContainer(ctx, ollamaContainer.ID); err != nil {
			return fmt.Errorf("start container: %w", err)
		}

		time.Sleep(2 * time.Second)
	}

	return nil
}

func (o *OllamaClient) waitForAPI(ctx context.Context) error {
	maxAttempts := 30
	delay := 500 * time.Millisecond

	for i := 0; i < maxAttempts; i++ {
		if err := o.Ping(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("ollama API not available after %d attempts", maxAttempts)
}

func (o *OllamaClient) BaseURL() string {
	return o.baseURL
}
