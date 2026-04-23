package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dev-cli/internal/config"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAIProvider implements Provider against any OpenAI-compatible endpoint:
// OpenAI itself, OpenRouter, Groq, Together, DeepInfra, local vLLM, etc.
// The caller supplies base URL, key, and default model via Config.
type OpenAIProvider struct {
	client  *openai.Client
	model   string
	baseURL string
}

var _ Provider = (*OpenAIProvider)(nil)

// NewOpenAIProvider constructs a provider from config. Returns nil when no API
// key is configured — callers can treat nil as "unconfigured" without
// checking each field.
func NewOpenAIProvider(cfg *config.Config) *OpenAIProvider {
	if cfg == nil || cfg.OpenAIKey == "" {
		return nil
	}

	baseURL := strings.TrimRight(cfg.OpenAIURL, "/") + "/"
	if cfg.OpenAIURL == "" {
		baseURL = "https://api.openai.com/v1/"
	}

	model := cfg.OpenAIModel
	if model == "" {
		model = "gpt-4o-mini"
	}

	client := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(cfg.OpenAIKey),
		option.WithHTTPClient(&http.Client{Timeout: RequestTimeout}),
	)

	return &OpenAIProvider{
		client:  &client,
		model:   model,
		baseURL: baseURL,
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

// DefaultModel returns the configured default model (used by the routing layer
// to pick the right model string when the caller asks for "whatever the cloud
// provider prefers").
func (p *OpenAIProvider) DefaultModel() string { return p.model }

// ChatCompletion delegates to the shared SDK helper. If the caller did not
// specify a model, the provider's configured default is used.
func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if strings.TrimSpace(req.Model) == "" {
		req.Model = p.model
	}
	resp, err := chatCompletionViaSDK(ctx, p.client, req)
	if err != nil {
		return nil, fmt.Errorf("openai chat completion: %w", err)
	}
	return resp, nil
}

// Ping performs a cheap reachability check against the /models endpoint.
// Returns nil when the endpoint answers 2xx within the given timeout.
func (p *OpenAIProvider) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := p.client.Models.List(ctx)
	return err
}
