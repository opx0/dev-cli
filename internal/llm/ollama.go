package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DefaultOllamaURL = "http://localhost:11434"
	DefaultModel     = "qwen2.5-coder:3b-instruct"
	FallbackModel    = "deepseek-r1:1.5b"
	RequestTimeout   = 30 * time.Second
)

type ExplainResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}

type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewClient() *Client {
	baseURL := DefaultOllamaURL
	if envURL := os.Getenv("DEV_CLI_OLLAMA_URL"); envURL != "" {
		baseURL = envURL
	}

	model := DefaultModel
	if envModel := os.Getenv("DEV_CLI_OLLAMA_MODEL"); envModel != "" {
		model = envModel
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
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
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
   - If no fix possible, refer to sources more authetic to that problem to precise documentation etc ""

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
