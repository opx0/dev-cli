package llm

import (
	"bytes"
	"dev-cli/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	DefaultOllamaURL = "http://localhost:11434"
	DefaultModel     = "qwen2.5-coder:3b-instruct"
	FallbackModel    = "qwen2.5-coder:3b-instruct-q8_0"
	RequestTimeout   = 30 * time.Second
)

func EnsureOllamaRunning() error {

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(DefaultOllamaURL + "/api/tags")
	if err == nil {
		resp.Body.Close()
		return nil
	}

	fmt.Println("\033[33m⚡ Ollama not running, starting...\033[0m")

	startCmd := exec.Command("docker", "start", "ollama")
	if err := startCmd.Run(); err == nil {

		return waitForOllama(client, 30*time.Second)
	}

	fmt.Println("\033[90m  Creating Ollama container...\033[0m")
	createCmd := exec.Command("docker", "run", "-d",
		"--name", "ollama",
		"-p", "11434:11434",
		"-v", "ollama:/root/.ollama",
		"--restart", "unless-stopped",
		"ollama/ollama")

	output, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create Ollama container: %w\n%s", err, string(output))
	}

	return waitForOllama(client, 60*time.Second)
}

func waitForOllama(client *http.Client, timeout time.Duration) error {
	start := time.Now()
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for Ollama to start")
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

type ExplainResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}

type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	baseURL := DefaultOllamaURL
	if cfg.OllamaURL != "" {
		baseURL = cfg.OllamaURL
	}

	model := DefaultModel
	if cfg.OllamaModel != "" {
		model = cfg.OllamaModel
	}

	return &Client{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

type generateRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	Stream    bool   `json:"stream"`
	Format    string `json:"format,omitempty"`
	KeepAlive string `json:"keep_alive,omitempty"`
}

type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c *Client) Explain(cmd string, exitCode int, output string) (*ExplainResult, error) {
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

	req := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	if os.Getenv("DEV_CLI_OLLAMA_UNLOAD") == "true" {
		req.KeepAlive = "0m"
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var result ExplainResult
	responseText := strings.TrimSpace(genResp.Response)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return &ExplainResult{Explanation: responseText, Fix: ""}, nil
	}

	return &result, nil
}

func (c *Client) Research(query string) (*ResearchResult, error) {
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

	req := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	if os.Getenv("DEV_CLI_OLLAMA_UNLOAD") == "true" {
		req.KeepAlive = "0m"
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	responseText := strings.TrimSpace(genResp.Response)

	var result ResearchResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("parse solutions: %w", err)
	}

	result.Query = query
	return &result, nil
}

func (c *Client) AnalyzeLog(logLines string) (*LogAnalysisResult, error) {
	prompt := fmt.Sprintf(`You are a Log Analyzer. Identify the error in these log lines.

OUTPUT JSON ONLY:
{
  "explanation": "Brief description of the error (1 sentence)",
  "fix": "Suggested command or action to fix it (or empty if unknown)"
}

LOGS:
%s`, logLines)

	req := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}

	if os.Getenv("DEV_CLI_OLLAMA_UNLOAD") == "true" {
		req.KeepAlive = "0m"
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var result LogAnalysisResult
	responseText := strings.TrimSpace(genResp.Response)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return &LogAnalysisResult{Explanation: responseText}, nil
	}

	return &result, nil
}

func (c *Client) Solve(goal string) (string, error) {
	prompt := fmt.Sprintf(`You are an Autonomous CLI Agent. The user wants to: "%s".
Provide a SINGLE shell command to achieve this.

RULES:
1. Output ONLY the command. No markdown, no explanations.
2. If multiple steps are needed, chain them with && or ;
3. Assume a standard Linux environment.
4. BE SAFE. Do not return commands that delete data without confirmation unless explicitly asked.

GOAL: %s
COMMAND:`, goal, goal)

	req := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	if os.Getenv("DEV_CLI_OLLAMA_UNLOAD") == "true" {
		req.KeepAlive = "0m"
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return strings.TrimSpace(genResp.Response), nil
}

// ToolCallResult represents the result of a tool-aware LLM generation.
type ToolCallResult struct {
	ToolName   string         `json:"tool_name"`
	Parameters map[string]any `json:"parameters"`
	Reasoning  string         `json:"reasoning,omitempty"`
}

// GenerateWithTools calls the LLM with tool definitions and expects a tool call response.
func (c *Client) GenerateWithTools(prompt string, toolSchemas string) (*ToolCallResult, error) {
	systemPrompt := fmt.Sprintf(`You are an AI assistant with access to tools. Based on the user's request, determine which tool to use and with what parameters.

AVAILABLE TOOLS:
%s

RULES:
1. Analyze the user's request carefully
2. Select the most appropriate tool
3. Determine the correct parameters
4. Respond with ONLY valid JSON in this exact format:
{
  "tool_name": "name_of_tool",
  "parameters": {
    "param1": "value1",
    "param2": "value2"
  },
  "reasoning": "brief explanation of why this tool was chosen"
}

Do NOT include any text outside the JSON object.`, toolSchemas)

	fullPrompt := fmt.Sprintf(`%s

USER REQUEST: %s

JSON RESPONSE:`, systemPrompt, prompt)

	req := generateRequest{
		Model:  c.model,
		Prompt: fullPrompt,
		Stream: false,
		Format: "json",
	}

	if os.Getenv("DEV_CLI_OLLAMA_UNLOAD") == "true" {
		req.KeepAlive = "0m"
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/generate", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var result ToolCallResult
	responseText := strings.TrimSpace(genResp.Response)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("parse tool call: %w (response: %s)", err, responseText)
	}

	return &result, nil
}
