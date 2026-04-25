package llm

import (
	"bytes"
	"context"
	"dev-cli/internal/config"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
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

// ── Docker Management ────────────────────────────────────────────────────────

// OllamaStatus describes the state of the Ollama service for diagnostics.
type OllamaStatus struct {
	Running    bool
	URL        string
	ModelCount int
	ViaDocker  bool
}

// CheckOllamaStatus probes the Ollama API and returns its status without
// attempting to start anything. Useful for offline-mode indicators.
func CheckOllamaStatus() OllamaStatus {
	status := OllamaStatus{URL: DefaultOllamaURL}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(DefaultOllamaURL + "/api/tags")
	if err != nil {
		return status
	}
	defer resp.Body.Close()

	status.Running = true

	var tags struct {
		Models []struct{ Name string } `json:"models"`
	}
	if json.NewDecoder(resp.Body).Decode(&tags) == nil {
		status.ModelCount = len(tags.Models)
	}

	// Detect if running via Docker
	dockerCheck := exec.Command("docker", "ps", "--filter", "name=ollama", "--format", "{{.Names}}")
	if out, err := dockerCheck.Output(); err == nil && strings.TrimSpace(string(out)) != "" {
		status.ViaDocker = true
	}

	return status
}

func EnsureOllamaRunning() error {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(DefaultOllamaURL + "/api/tags")
	if err == nil {
		resp.Body.Close()
		return nil
	}

	fmt.Println("\033[33m⚡ Ollama not running, attempting to start...\033[0m")

	// Check if Docker is available first
	if err := exec.Command("docker", "info").Run(); err != nil {
		return fmt.Errorf("ollama not running and Docker is unavailable\n" +
			"  \033[36mFix options:\033[0m\n" +
			"  1. Start Docker:  \033[1msudo systemctl start docker\033[0m\n" +
			"  2. Install Ollama natively:  \033[1mcurl -fsSL https://ollama.com/install.sh | sh\033[0m\n" +
			"  3. Run full setup:  \033[1mmake setup\033[0m")
	}

	// Try starting existing container
	startCmd := exec.Command("docker", "start", "ollama")
	if err := startCmd.Run(); err == nil {
		fmt.Println("\033[90m  Restarting existing Ollama container...\033[0m")
		return waitForOllama(client, 30*time.Second)
	}

	// Try creating a new container
	fmt.Println("\033[90m  Creating new Ollama container...\033[0m")
	createCmd := exec.Command("docker", "run", "-d",
		"--name", "ollama",
		"-p", "11434:11434",
		"-v", "ollama:/root/.ollama",
		"--restart", "unless-stopped",
		"ollama/ollama")

	output, createErr := createCmd.CombinedOutput()
	if createErr != nil {
		outStr := strings.TrimSpace(string(output))
		if strings.Contains(outStr, "already in use") {
			// Container exists but is stopped with a different state
			_ = exec.Command("docker", "rm", "-f", "ollama").Run()
			output, createErr = createCmd.CombinedOutput()
		}
		if createErr != nil {
			return fmt.Errorf("failed to create Ollama container: %w\n  Output: %s\n"+
				"  \033[36mTry:\033[0m make setup", createErr, strings.TrimSpace(string(output)))
		}
	}

	return waitForOllama(client, 60*time.Second)
}

func waitForOllama(client *http.Client, timeout time.Duration) error {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("Ollama did not become ready within %s\n"+
				"  \033[36mDiagnostics:\033[0m\n"+
				"  • Check container logs: \033[1mdocker logs ollama\033[0m\n"+
				"  • Check container status: \033[1mdocker ps -a --filter name=ollama\033[0m\n"+
				"  • Restart from scratch: \033[1mmake setup\033[0m", timeout)
		}

		resp, err := client.Get(DefaultOllamaURL + "/api/tags")
		if err == nil {
			resp.Body.Close()
			fmt.Println("\033[32m✓ Ollama is ready\033[0m")
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}
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
	client  *openai.Client
	model   string
	baseURL string
	cfg     *config.Config
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
		client:  &client,
		model:   model,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		cfg:     cfg,
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
		model := strings.TrimSpace(req.Model)
		if model == "" {
			model = p.model
		}

		if (isModelNotFoundError(err) || isRequestTimeoutError(err)) && model != "" {
			if pullErr := p.PullModel(ctx, model); pullErr != nil {
				return nil, fmt.Errorf("ollama model '%s' not found and auto-pull failed: %w", model, pullErr)
			}

			completion, err = p.client.Chat.Completions.New(ctx, params)
			if err == nil {
				goto completionReady
			}
		}

		return nil, fmt.Errorf("ollama chat completion: %w", err)
	}

completionReady:

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

func isModelNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "model") &&
		strings.Contains(msg, "not found")
}

func isRequestTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "client.timeout exceeded") ||
		strings.Contains(msg, "timeout")
}

// PullModel downloads an Ollama model from the configured registry. Exported
// so health checks / doctor --fix can trigger a pull when the configured model
// is missing.
func (p *OllamaProvider) PullModel(ctx context.Context, model string) error {
	payload := struct {
		Name   string `json:"name"`
		Stream bool   `json:"stream"`
	}{
		Name:   model,
		Stream: true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal pull request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pull model '%s': %w", model, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("pull model '%s' failed with status %d: %s", model, resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	fmt.Printf("\n\033[36mℹ\033[0m Auto-pulling model '%s' (this may take a few minutes)...\n", model)
	
	decoder := json.NewDecoder(resp.Body)
	for {
		var progress struct {
			Status    string `json:"status"`
			Completed int64  `json:"completed"`
			Total     int64  `json:"total"`
		}
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode progress: %w", err)
		}
		
		if progress.Total > 0 {
			percent := float64(progress.Completed) / float64(progress.Total) * 100
			fmt.Printf("\r\033[K  \033[90m%s\033[0m %.1f%% (%d/%d MB)", progress.Status, percent, progress.Completed/(1024*1024), progress.Total/(1024*1024))
		} else {
			fmt.Printf("\r\033[K  \033[90m%s\033[0m", progress.Status)
		}
	}
	fmt.Println()

	return nil
}

// ── Convenience Methods (for cmd/ backward compatibility) ────────────────────

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

	messages := []Message{UserMsg(prompt)}
	if p.cfg.OllamaUnload {
		// Note: keep_alive is Ollama-specific and not supported via OpenAI API.
		// The model will use Ollama's default keep_alive setting.
	}

	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Model:    p.model,
		Messages: messages,
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

// AnalyzeLog identifies errors in log lines.
func (p *OllamaProvider) AnalyzeLog(logLines string) (*LogAnalysisResult, error) {
	prompt := fmt.Sprintf(`You are a Log Analyzer. Identify the error in these log lines.

OUTPUT JSON ONLY:
{
  "explanation": "Brief description of the error (1 sentence)",
  "fix": "Suggested command or action to fix it (or empty if unknown)"
}

LOGS:
%s`, logLines)

	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Model:    p.model,
		Messages: []Message{UserMsg(prompt)},
		Format:   "json",
	})
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}

	var result LogAnalysisResult
	responseText := strings.TrimSpace(resp.Content)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return &LogAnalysisResult{Explanation: responseText}, nil
	}

	return &result, nil
}

// Solve generates a single shell command to achieve a goal.
func (p *OllamaProvider) Solve(goal string) (string, error) {
	prompt := fmt.Sprintf(`You are an Autonomous CLI Agent. The user wants to: "%s".
Provide a SINGLE shell command to achieve this.

RULES:
1. Output ONLY the command. No markdown, no explanations.
2. If multiple steps are needed, chain them with && or ;
3. Assume a standard Linux environment.
4. BE SAFE. Do not return commands that delete data without confirmation unless explicitly asked.

GOAL: %s
COMMAND:`, goal, goal)

	resp, err := p.ChatCompletion(context.Background(), ChatRequest{
		Model:    p.model,
		Messages: []Message{UserMsg(prompt)},
	})
	if err != nil {
		return "", fmt.Errorf("call Ollama: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
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

// ── Legacy Compatibility ─────────────────────────────────────────────────────

// Client is a backward-compatible alias. New code should use OllamaProvider.
type Client = OllamaProvider

// NewClient creates a new OllamaProvider (backward-compatible name).
func NewClient(cfg *config.Config) *OllamaProvider {
	return NewOllamaProvider(cfg)
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
