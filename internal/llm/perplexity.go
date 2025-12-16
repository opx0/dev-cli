package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dev-cli/internal/config"
)

const (
	PerplexityAPIURL = "https://api.perplexity.ai/chat/completions"
)

type PerplexityClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewPerplexityClient(cfg *config.Config) *PerplexityClient {
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

func (c *PerplexityClient) Research(ctx context.Context, query string) (*ResearchResult, error) {
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
