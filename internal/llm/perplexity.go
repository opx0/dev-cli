package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dev-cli/internal/config"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

const (
	PerplexityAPIURL = "https://api.perplexity.ai"
)

// ── PerplexityProvider ───────────────────────────────────────────────────────

// PerplexityProvider implements the Provider interface using Perplexity's
// OpenAI-compatible chat completions API.
type PerplexityProvider struct {
	client *openai.Client
	model  string
}

var _ Provider = (*PerplexityProvider)(nil)

// NewPerplexityProvider creates a new PerplexityProvider.
// Returns nil if no API key is configured.
func NewPerplexityProvider(cfg *config.Config) *PerplexityProvider {
	if cfg.PerplexityKey == "" {
		return nil
	}

	client := openai.NewClient(
		option.WithBaseURL(PerplexityAPIURL+"/"),
		option.WithAPIKey(cfg.PerplexityKey),
		option.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
	)

	return &PerplexityProvider{
		client: &client,
		model:  cfg.PerplexityModel,
	}
}

func (p *PerplexityProvider) Name() string { return "perplexity" }

// ChatCompletion sends a chat completion request through the OpenAI SDK to Perplexity.
func (p *PerplexityProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req = sanitizeCloudRequest(req)
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, convertMessage(msg))
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(req.Model),
		Messages: messages,
	}

	if len(req.Tools) > 0 {
		tools := make([]openai.ChatCompletionToolParam, 0, len(req.Tools))
		for _, td := range req.Tools {
			tools = append(tools, openai.ChatCompletionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        td.Name,
					Description: param.NewOpt(td.Description),
					Parameters:  shared.FunctionParameters(td.Parameters),
				},
			})
		}
		params.Tools = tools
	}

	if req.Temperature != nil {
		params.Temperature = param.NewOpt(*req.Temperature)
	}

	if req.MaxTokens != nil {
		params.MaxCompletionTokens = param.NewOpt(*req.MaxTokens)
	}

	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("perplexity chat completion: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("no response from Perplexity")
	}

	choice := completion.Choices[0]

	var toolCalls []ToolCall
	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return &ChatResponse{
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     completion.Usage.PromptTokens,
			CompletionTokens: completion.Usage.CompletionTokens,
			TotalTokens:      completion.Usage.TotalTokens,
		},
	}, nil
}

// ── Convenience Methods ──────────────────────────────────────────────────────

// Research provides web-enhanced step-by-step solutions for a query.
func (p *PerplexityProvider) Research(ctx context.Context, query string) (*ResearchResult, error) {
	prompt := fmt.Sprintf(`You are a Senior Developer Assistant. The user needs to: "%s".
Provide the TOP 3 distinct ways to achieve this.

RULES:
1. Option 1 = "Best Practice" / Modern way
2. Option 2 = "Quickest/Easiest" way
3. Option 3 = "Alternative" (edge case or manual approach)
4. Each solution can have multiple steps
5. Step type is "command" for shell commands, "file" for code snippets to add to files
6. For "file" type, include the target filename in "file" field
7. Include source URLs when available

OUTPUT JSON ONLY (No markdown, no code fences):
{
  "solutions": [
    {
      "id": 1,
      "title": "Using npm (Recommended)",
      "description": "Modern package manager with better caching",
      "steps": [
        {"type": "command", "content": "npm install tailwindcss", "note": "Install package"},
        {"type": "command", "content": "npx tailwindcss init", "note": "Initialize config"},
        {"type": "file", "file": "tailwind.config.js", "content": "module.exports = { content: ['./src/**/*.{js,jsx}'] }", "note": "Configure paths"}
      ],
      "source": "https://tailwindcss.com/docs"
    }
  ]
}`, query)

	resp, err := p.ChatCompletion(ctx, ChatRequest{
		Model: p.model,
		Messages: []Message{
			SystemMsg("You are a helpful developer assistant. Always respond with valid JSON only, no markdown formatting."),
			UserMsg(prompt),
		},
	})
	if err != nil {
		return nil, err
	}

	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var result ResearchResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse solutions: %w", err)
	}

	result.Query = query
	return &result, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
