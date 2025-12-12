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

// Default configuration
const (
	DefaultOllamaURL = "http://localhost:11434"
	DefaultModel     = "qwen2.5-coder:3b-instruct"
	FallbackModel    = "deepseek-r1:1.5b"
	RequestTimeout   = 30 * time.Second
)

// ExplainResult contains the parsed LLM response
type ExplainResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}

// Client handles Ollama API communication
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient creates an Ollama client
// Uses DEV_CLI_OLLAMA_URL env var if set, otherwise default
// Uses DEV_CLI_OLLAMA_MODEL env var if set, otherwise qwen2.5-coder:3b-instruct
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

// generateRequest is the Ollama API request format
type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
}

// generateResponse is the Ollama API response format
type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Explain sends a prompt to Ollama and returns the explanation
func (c *Client) Explain(cmd string, exitCode int, output string) (*ExplainResult, error) {
	// Truncate output if too long
	if len(output) > 2000 {
		output = output[len(output)-2000:]
	}

	prompt := fmt.Sprintf(`You are a CLI error analyzer. Analyze this failed command and respond with JSON only.

RULES:
1. "explanation" = Brief 1-sentence error cause
2. "fix" = EXACT shell command to run (NOT advice, NOT instructions - just the command)
   - Good fix: "npm init -y"
   - Bad fix: "Make sure package.json exists"
   - If no fix possible, use empty string ""

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
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/api/generate",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse the JSON response from the LLM
	var result ExplainResult
	responseText := strings.TrimSpace(genResp.Response)
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		// If JSON parsing fails, use raw response as explanation
		return &ExplainResult{
			Explanation: responseText,
			Fix:         "",
		}, nil
	}

	return &result, nil
}
