package llm

import (
	"context"
	"dev-cli/internal/config"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

const (
	DefaultOllamaURL = "http://localhost:11434"
	DefaultModel     = "smallthinker"
	FallbackModel    = "smallthinker"
	RequestTimeout   = 5 * time.Minute
)

// CheckOllamaAvailable verifies the configured local provider without changing
// Docker, processes, or any other machine state.
func CheckOllamaAvailable(cfg *config.Config) error {
	baseURL := DefaultOllamaURL
	if cfg != nil && cfg.OllamaURL != "" {
		baseURL = strings.TrimSuffix(cfg.OllamaURL, "/")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("create ollama request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama is unreachable at %s: %w", baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama at %s returned status %d", baseURL, resp.StatusCode)
	}
	return nil
}

// ── Response Types (kept for cmd/ compatibility) ─────────────────────────────

type ExplainResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}

type CheatSheetResult struct {
	Prerequisites []string        `json:"prerequisites"`
	Commands      []CheatSheetCmd `json:"commands"`
}

type CheatSheetCmd struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// ── OllamaProvider ───────────────────────────────────────────────────────────

// OllamaProvider implements the Provider interface using the OpenAI-compatible
// API that Ollama serves at /v1/.
type OllamaProvider struct {
	client *openai.Client
	model  string
}

var _ Provider = (*OllamaProvider)(nil)

// NewOllamaProvider creates a new OllamaProvider backed by the openai-go SDK.
func NewOllamaProvider(cfg *config.Config) *OllamaProvider {
	baseURL := DefaultOllamaURL
	if cfg.OllamaURL != "" {
		baseURL = cfg.OllamaURL
	}

	model := DefaultModel
	if cfg.OllamaModel != "" {
		model = cfg.OllamaModel
	}

	// Ollama serves an OpenAI-compatible API at /v1/
	client := openai.NewClient(
		option.WithBaseURL(baseURL+"/v1/"),
		option.WithAPIKey("ollama"), // Ollama ignores auth but the SDK requires a value
		option.WithHTTPClient(&http.Client{Timeout: RequestTimeout}),
	)

	return &OllamaProvider{
		client: &client,
		model:  model,
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

// ChatCompletion sends a chat completion request through the OpenAI SDK to Ollama.
func (p *OllamaProvider) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Build messages
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, convertMessage(msg))
	}

	// Build params
	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(req.Model),
		Messages: messages,
	}

	// Add tools if provided
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

	// Call the API
	completion, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("ollama chat completion: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("ollama returned no choices")
	}

	choice := completion.Choices[0]

	// Convert tool calls
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

// ── Command-oriented methods ─────────────────────────────────────────────────

// Explain analyzes a failed command and returns an explanation with optional fix.
func (p *OllamaProvider) Explain(cmd string, exitCode int, output string) (*ExplainResult, error) {
	if len(output) > 2000 {
		output = output[len(output)-2000:]
	}

	prompt := fmt.Sprintf(`You are a CLI error analyzer. Analyze this failed command and respond with JSON only.

RULES:
1. "explanation" = Brief 1-sentence error cause can attend for more precision only if needed.
2. "fix" = EXACT shell command to run (NOT advice, NOT instructions - just the command)
   - Good fix: "npm init -y new line and more command if needed to run in sequence"
   - Bad fix: "Make sure package.json exists"
   - If no fix possible, refer to sources more authentic to that problem to precise documentation etc ""

EXAMPLES:
- package.json missing → {"explanation": "Missing package.json", "fix": "npm init -y"}
- permission denied → {"explanation": "Permission denied", "fix": "sudo !!"}
- command not found → {"explanation": "Command not installed", "fix": ""}

Command: %s
Exit Code: %d
Output: %s

JSON response:`, cmd, exitCode, output)

	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Model:    p.model,
		Messages: []Message{UserMsg(prompt)},
		Format:   "json",
	})
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}

	var result ExplainResult
	responseText := strings.TrimSpace(resp.Content)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return &ExplainResult{Explanation: responseText, Fix: ""}, nil
	}

	return &result, nil
}

// Research provides step-by-step solutions for a query.
func (p *OllamaProvider) Research(query string) (*ResearchResult, error) {
	prompt := fmt.Sprintf(`You are a Senior Developer Assistant. The user needs to: "%s".
Provide the TOP 3 distinct ways to achieve this.

RULES:
1. Option 1 = "Best Practice" / Modern way
2. Option 2 = "Quickest/Easiest" way
3. Option 3 = "Alternative" (edge case or manual approach)
4. Each solution can have multiple steps
5. Step type is "command" for shell commands, "file" for code snippets
6. For "file" type, include the target filename in "file" field

OUTPUT JSON ONLY:
{
  "solutions": [
    {
      "id": 1,
      "title": "Using Docker (Recommended)",
      "description": "Isolated environment",
      "steps": [
        {"type": "command", "content": "docker run -d postgres", "note": "Start container"}
      ],
      "source": ""
    }
  ]
}`, query)

	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Model:    p.model,
		Messages: []Message{UserMsg(prompt)},
		Format:   "json",
	})
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}

	responseText := strings.TrimSpace(resp.Content)
	var result ResearchResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("parse solutions: %w", err)
	}

	result.Query = query
	return &result, nil
}

// CheatSheet generates useful shell commands for a tool/topic.
func (p *OllamaProvider) CheatSheet(tool, topic string, count int) (*CheatSheetResult, error) {
	query := tool
	if topic != "important and commonly used" {
		query = tool + " " + topic
	}

	prompt := fmt.Sprintf(`Give me %d useful shell commands for: "%s"

Include "prerequisites" array with package install commands if special packages are needed.

JSON format:
{"prerequisites":["sudo pacman -S ntfs-3g"],"commands":[{"command":"sudo mount -t ntfs-3g /dev/sda1 /mnt","description":"Mount NTFS partition"}]}

Commands for "%s":`, count, query, query)

	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Model:    p.model,
		Messages: []Message{UserMsg(prompt)},
		Format:   "json",
	})
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}

	var result CheatSheetResult
	responseText := strings.TrimSpace(resp.Content)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return &CheatSheetResult{
			Commands: []CheatSheetCmd{{Command: responseText, Description: "raw response"}},
		}, nil
	}

	return &result, nil
}

// ── SDK Helpers ──────────────────────────────────────────────────────────────

// convertMessage converts our Message type to the openai SDK union type.
func convertMessage(msg Message) openai.ChatCompletionMessageParamUnion {
	switch msg.Role {
	case "system":
		return openai.SystemMessage(msg.Content)
	case "user":
		return openai.UserMessage(msg.Content)
	case "assistant":
		if len(msg.ToolCalls) > 0 {
			sdkCalls := make([]openai.ChatCompletionMessageToolCallParam, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				sdkCalls[i] = openai.ChatCompletionMessageToolCallParam{
					ID: tc.ID,
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
			asstMsg := openai.AssistantMessage(msg.Content)
			if asstMsg.OfAssistant != nil {
				asstMsg.OfAssistant.ToolCalls = sdkCalls
			}
			return asstMsg
		}
		return openai.AssistantMessage(msg.Content)
	case "tool":
		return openai.ToolMessage(msg.Content, msg.ToolCallID)
	default:
		return openai.UserMessage(msg.Content)
	}
}
