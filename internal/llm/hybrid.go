package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"dev-cli/internal/config"
)

var webKeywords = []string{
	"install",
	"latest",
	"version",
	"how to",
	"compare",
	"why",
	"best",
	"setup",
	"configure",
	"deploy",
	"update",
	"upgrade",
}

type HybridClient struct {
	perplexity *PerplexityClient
	ollama     *Client
}

func NewHybridClient() *HybridClient {
	cfg := config.Load()
	return &HybridClient{
		perplexity: NewPerplexityClient(cfg),
		ollama:     NewClient(cfg),
	}
}

func (h *HybridClient) Research(query string) (*ResearchResult, error) {
	if h.perplexity != nil && needsWebSearch(query) {
		result, err := h.perplexity.Research(context.Background(), query)
		if err == nil {
			return result, nil
		}
	}

	return h.ollama.Research(query)
}

func (h *HybridClient) HasPerplexity() bool {
	return h.perplexity != nil
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
