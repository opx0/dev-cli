package ai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"dev-cli/internal/core"
)

const (
	DefaultOllamaURL = "http://localhost:11434"
	DefaultModel     = "qwen2.5-coder:3b-instruct"
	FallbackModel    = "qwen2.5-coder:3b-instruct-q8_0"
	RequestTimeout   = 30 * time.Second
	PerplexityAPIURL = "https://api.perplexity.ai/chat/completions"
)

type ExplainResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}

type Step struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	File    string `json:"file,omitempty"`
	Note    string `json:"note,omitempty"`
}

type Solution struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Steps       []Step `json:"steps"`
	Source      string `json:"source,omitempty"`
}

type ResearchResult struct {
	Query     string     `json:"query"`
	Solutions []Solution `json:"solutions"`
}

type LogAnalysisResult struct {
	Explanation string `json:"explanation"`
	Fix         string `json:"fix"`
}

type ToolCallResult struct {
	ToolName   string         `json:"tool_name"`
	Parameters map[string]any `json:"parameters"`
	Reasoning  string         `json:"reasoning,omitempty"`
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

type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllamaClient(cfg *core.Config) *OllamaClient {
	baseURL := DefaultOllamaURL
	if cfg.OllamaURL != "" {
		baseURL = cfg.OllamaURL
	}

	model := DefaultModel
	if cfg.OllamaModel != "" {
		model = cfg.OllamaModel
	}

	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: RequestTimeout,
		},
	}
}

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

func (c *OllamaClient) Explain(cmd string, exitCode int, output string) (*ExplainResult, error) {
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

	return c.generateExplain(prompt)
}

func (c *OllamaClient) generateExplain(prompt string) (*ExplainResult, error) {
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

func (c *OllamaClient) Research(query string) (*ResearchResult, error) {
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

func (c *OllamaClient) AnalyzeLog(logLines string) (*LogAnalysisResult, error) {
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

func (c *OllamaClient) Solve(goal string) (string, error) {
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

func (c *OllamaClient) GenerateWithTools(prompt string, toolSchemas string) (*ToolCallResult, error) {
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

type perplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type perplexityRequest struct {
	Model    string              `json:"model"`
	Messages []perplexityMessage `json:"messages"`
}

type perplexityChoice struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

type perplexityResponse struct {
	Choices []perplexityChoice `json:"choices"`
}

type PerplexityClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewPerplexityClient(cfg *core.Config) *PerplexityClient {
	if cfg.PerplexityKey == "" {
		return nil
	}

	return &PerplexityClient{
		apiKey: cfg.PerplexityKey,
		model:  cfg.PerplexityModel,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *PerplexityClient) Research(ctx context.Context, query string) (*ResearchResult, error) {
	prompt := fmt.Sprintf(`You are a Senior Developer Assistant. The user needs to: "%s".
Provide the TOP 3 distinct ways to achieve this.

OUTPUT JSON ONLY (No markdown, no code fences):
{
  "solutions": [
    {
      "id": 1,
      "title": "Using npm (Recommended)",
      "description": "Modern package manager with better caching",
      "steps": [
        {"type": "command", "content": "npm install tailwindcss", "note": "Install package"}
      ],
      "source": "https://tailwindcss.com/docs"
    }
  ]
}`, query)

	reqBody, err := json.Marshal(perplexityRequest{
		Model: c.model,
		Messages: []perplexityMessage{
			{Role: "system", Content: "You are a helpful developer assistant. Always respond with valid JSON only, no markdown formatting."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", PerplexityAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Perplexity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("perplexity status %d: %s", resp.StatusCode, string(body))
	}

	var pResp perplexityResponse
	if err := json.NewDecoder(resp.Body).Decode(&pResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(pResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Perplexity")
	}

	content := strings.TrimSpace(pResp.Choices[0].Message.Content)
	content = stripMarkdownFences(content)

	var result ResearchResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("parse solutions: %w", err)
	}

	result.Query = query
	return &result, nil
}

func (c *PerplexityClient) AnalyzeLog(ctx context.Context, logLines string) (*LogAnalysisResult, error) {
	prompt := fmt.Sprintf(`You are a Log Analyzer. Identify the error in these log lines.

OUTPUT JSON ONLY (No markdown):
{
  "explanation": "Brief description of the error (1 sentence)",
  "fix": "Suggested command or action to fix it (or empty if unknown)"
}

LOGS:
%s`, logLines)

	reqBody, err := json.Marshal(perplexityRequest{
		Model: c.model,
		Messages: []perplexityMessage{
			{Role: "system", Content: "You are a helpful developer assistant. Always respond with valid JSON only, no markdown formatting."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", PerplexityAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Perplexity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("perplexity status %d: %s", resp.StatusCode, string(body))
	}

	var pResp perplexityResponse
	if err := json.NewDecoder(resp.Body).Decode(&pResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(pResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Perplexity")
	}

	content := strings.TrimSpace(pResp.Choices[0].Message.Content)
	content = stripMarkdownFences(content)

	var result LogAnalysisResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return &LogAnalysisResult{Explanation: content}, nil
	}

	return &result, nil
}

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

var webKeywords = []string{
	"install", "latest", "version", "how to", "compare",
	"why", "best", "setup", "configure", "deploy", "update", "upgrade",
}

type cacheEntry struct {
	result    *ResearchResult
	timestamp time.Time
}

type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	keys    []string
	maxSize int
	ttl     time.Duration
}

func NewResponseCache(maxSize int, ttl time.Duration) *ResponseCache {
	return &ResponseCache{
		entries: make(map[string]cacheEntry),
		keys:    make([]string, 0),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

func hashQuery(query string) string {
	h := sha256.New()
	h.Write([]byte(strings.ToLower(strings.TrimSpace(query))))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func (c *ResponseCache) Get(query string) (*ResearchResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := hashQuery(query)
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Since(entry.timestamp) > c.ttl {
		return nil, false
	}

	return entry.result, true
}

func (c *ResponseCache) Set(query string, result *ResearchResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := hashQuery(query)

	if _, exists := c.entries[key]; exists {
		c.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
		c.moveToEnd(key)
		return
	}

	if len(c.keys) >= c.maxSize {
		oldest := c.keys[0]
		delete(c.entries, oldest)
		c.keys = c.keys[1:]
	}

	c.entries[key] = cacheEntry{result: result, timestamp: time.Now()}
	c.keys = append(c.keys, key)
}

func (c *ResponseCache) moveToEnd(key string) {
	for i, k := range c.keys {
		if k == key {
			c.keys = append(c.keys[:i], c.keys[i+1:]...)
			c.keys = append(c.keys, key)
			return
		}
	}
}

func (c *ResponseCache) Stats() (size int, capacity int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), c.maxSize
}

func (c *ResponseCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]cacheEntry)
	c.keys = make([]string, 0)
}

type HybridClient struct {
	perplexity *PerplexityClient
	ollama     *OllamaClient
	cache      *ResponseCache
}

var defaultCache = NewResponseCache(50, 10*time.Minute)

func NewHybridClient() *HybridClient {
	cfg := core.LoadConfig()
	return &HybridClient{
		perplexity: NewPerplexityClient(cfg),
		ollama:     NewOllamaClient(cfg),
		cache:      defaultCache,
	}
}

func (h *HybridClient) Research(query string) (*ResearchResult, error) {
	if cached, ok := h.cache.Get(query); ok {
		return cached, nil
	}

	var result *ResearchResult
	var err error

	if h.perplexity != nil && needsWebSearch(query) {
		result, err = h.perplexity.Research(context.Background(), query)
		if err == nil {
			h.cache.Set(query, result)
			return result, nil
		}
	}

	result, err = h.ollama.Research(query)
	if err == nil {
		h.cache.Set(query, result)
	}
	return result, err
}

func (h *HybridClient) HasPerplexity() bool {
	return h.perplexity != nil
}

func (h *HybridClient) CacheStats() (size int, capacity int) {
	return h.cache.Stats()
}

func (h *HybridClient) ClearCache() {
	h.cache.Clear()
}

func (h *HybridClient) AnalyzeLog(logLines string, aiMode string) (*LogAnalysisResult, error) {
	if os.Getenv("DEV_CLI_FORCE_LOCAL") != "" || aiMode == "local" {
		return h.ollama.AnalyzeLog(logLines)
	}

	if aiMode == "cloud" {
		if h.perplexity != nil {
			return h.perplexity.AnalyzeLog(context.Background(), logLines)
		}
		return nil, fmt.Errorf("cloud AI requested but PERPLEXITY_API_KEY is not set")
	}

	return h.ollama.AnalyzeLog(logLines)
}

func (h *HybridClient) Solve(goal string) (string, error) {
	return h.ollama.Solve(goal)
}

func needsWebSearch(query string) bool {
	if os.Getenv("DEV_CLI_FORCE_LOCAL") != "" {
		return false
	}

	lower := strings.ToLower(query)
	for _, kw := range webKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
